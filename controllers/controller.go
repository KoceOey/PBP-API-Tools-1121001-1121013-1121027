package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	redis "github.com/redis/go-redis/v9"
	"gopkg.in/gomail.v2"
)

var rdb *redis.Client
var context_redis = context.Background()

func RedisInit() {
	db := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	rdb = db
}

func CheckCron() {
	fmt.Println("halo")
}

func FailedHistoryCheck() {
	db := connect()
	defer db.Close()

	rows, err := db.Query("SELECT userid FROM failed_log GROUP BY userid")
	if err != nil {
		log.Println(err)
		return
	}

	var userid int
	var uid []int

	for rows.Next() {
		if err := rows.Scan(&userid); err != nil {
			log.Println(err)
			return
		} else {
			uid = append(uid, userid)
		}
	}

	for i, x := range uid {
		fmt.Println("loop ke-" + strconv.Itoa(i))
		res, err := db.Query("SELECT TIMESTAMPDIFF(minute,(SELECT time FROM failed_log WHERE userid = ? ORDER BY time DESC LIMIT 1),CURRENT_TIMESTAMP)", x)
		if err != nil {
			log.Println(err)
			return
		}

		var diff int

		for res.Next() {
			if err := res.Scan(&diff); err != nil {
				log.Println(err)
				return
			}
		}
		if diff >= 5 {
			DeleteFailedHistory(db, x)
			db.Exec("UPDATE users set state = 0 WHERE id = ?", x)
		}
	}
}

func GetRedis() string {

	val, err := rdb.Get(context_redis, "key").Result()
	if err == redis.Nil {
		log.Println(http.StatusNotFound, "data tidak ditemukan")
	} else if err != nil {
		log.Println(http.StatusBadRequest, "error get redis")
	}

	return val
}

func SendSuccessEmail(w http.ResponseWriter, r *http.Request, db *sql.DB, platform string) {
	mail := gomail.NewMessage()

	var user User
	err := json.Unmarshal([]byte(GetRedis()), &user)
	if err != nil {
		log.Println(http.StatusBadRequest, "error unmarshal redis")
	}

	log.Println("err : ")
	log.Println(err)

	mail.SetHeader("From", "hehehiha21@outlook.com")
	mail.SetHeader("To", user.Email)
	mail.SetHeader("Subject", "A New Log In")
	text := "Hello, " + user.Name + "! \nA new log in was made on " + platform
	mail.SetBody("text/plain", text)

	dialer := gomail.NewDialer("smtp-mail.outlook.com", 587, "hehehiha21@outlook.com", "Aw1kW0k!!")
	if err := dialer.DialAndSend(mail); err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func SendBlockedEmail(w http.ResponseWriter, r *http.Request, db *sql.DB, attempts []FailedAttempt) {
	mail := gomail.NewMessage()

	var user User
	err := json.Unmarshal([]byte(GetRedis()), &user)
	if err != nil {
		log.Println(http.StatusBadRequest, "error unmarshal redis")
	}

	log.Println("err : ")
	log.Println(err)

	mail.SetHeader("From", "hehehiha21@outlook.com")
	mail.SetHeader("To", user.Email)
	mail.SetHeader("Subject", "Account Blocked")
	text := "Hello, " + user.Name + "! Your account is block due to failed to log in 3 times, here are the login attempts, \n"
	for i := 0; i < len(attempts); i++ {
		text += "At " + attempts[i].Time + ", on " + attempts[i].Platform + "\n"
	}
	mail.SetBody("text/plain", text)

	dialer := gomail.NewDialer("smtp-mail.outlook.com", 587, "hehehiha21@outlook.com", "Aw1kW0k!!")
	if err := dialer.DialAndSend(mail); err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func DeleteFailedHistory(db *sql.DB, userid int) bool {
	_, errQuery := db.Exec("DELETE FROM failed_log WHERE userid =?", userid)
	if errQuery == nil {
		return true
	} else {
		fmt.Println(errQuery)
		return false
	}
}

func FailedLogin(w http.ResponseWriter, r *http.Request, db *sql.DB, user User, platform string) {
	// lihat login attempt
	attempts := CheckLoginAttempt(w, r, db, user)
	if attempts == nil {
		fmt.Println("Gagal mendapatkan history failed login")
		return
	} else if len(attempts) > 2 {
		go SendBlockedEmail(w, r, db, attempts)
		db.Exec("UPDATE users set state = 1 WHERE id = ?", user.Id)
		sendResponse(w, 400, "Wrong Email/Password!! Your account is now blocked")
		return
	}
	sendResponse(w, 400, "Wrong Email/Password!!")
}

func CheckLoginAttempt(w http.ResponseWriter, r *http.Request, db *sql.DB, user User) []FailedAttempt {
	rows, err := db.Query("select f.id, u.id, u.name, u.email, f.time, f.platform from failed_log f JOIN users u ON f.userid = u.id WHERE userid=?", user.Id)
	if err != nil {
		log.Println(err)
		return nil
	}

	var attempt FailedAttempt
	var attempts []FailedAttempt

	for rows.Next() {
		if err := rows.Scan(&attempt.Id, &attempt.User.Id, &attempt.User.Name, &attempt.User.Email, &attempt.Time, &attempt.Platform); err != nil {
			log.Println(err)
			return nil
		} else {
			attempts = append(attempts, attempt)
		}
	}
	return attempts
}

func UserLogin(w http.ResponseWriter, r *http.Request) {
	db := connect()
	defer db.Close()

	err := r.ParseForm()
	if err != nil {
		sendResponse(w, 400, "Failed")
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")
	header := r.Header.Get("platform")

	rows, err := db.Query("SELECT * FROM users WHERE email = ?", email)

	if err != nil {
		log.Println(err)
		sendResponse(w, 400, "Something went wrong, please try again.")
		return
	}

	var user User
	login := false

	// get user berdasarkan email
	for rows.Next() {
		if err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.Password, &user.State); err != nil {
			log.Println(err)
			sendResponse(w, 400, "Something went wrong!!")
			return
		} else {
			break
		}
	}

	reqRedis := User{
		Id:       user.Id,
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
		State:    user.State,
	}
	req, _ := json.Marshal(reqRedis)
	RedisInit()
	errSet := rdb.Set(context_redis, "key", req, 0).Err()

	if errSet != nil {
		log.Println("Error Set Redis", errSet)
	}

	// check account block/active
	if user.State == 1 {
		sendResponse(w, 400, "Your account is blocked because you have failed to login 3 times!")
		return
	}

	// check password
	if user.Password == password {
		login = true
	}

	// login failed
	if !login {
		db.Exec("INSERT INTO failed_log(userid,time,platform) values (?,CURRENT_TIMESTAMP,?)", user.Id, header)
		FailedLogin(w, r, db, user, header)
		return
	}

	go SendSuccessEmail(w, r, db, header)

	DeleteFailedHistory(db, user.Id)

	sendResponse(w, 200, "Success login from "+header)
}

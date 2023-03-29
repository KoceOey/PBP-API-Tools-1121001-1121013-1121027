package controllers

import (
	"fmt"
	"log"
	"net/http"
	"database/sql"
	//"strconv"
)

func DeleteFailedHistory(w http.ResponseWriter, r *http.Request, db *sql.DB, userid int) bool{
	_,errQuery := db.Exec("DELETE FROM failed_log WHERE userid =?",userid)
	if errQuery == nil {
		return true
	} else {
		fmt.Println(errQuery)
		return false
	}
}

func FailedLogin (w http.ResponseWriter, r *http.Request, db *sql.DB, userid int){
	if AddFailedLogin(w, r, db, userid){
		fmt.Println("Gagal menambahkan log failed")
	}

	// lihat login attempt
	attempts := CheckLoginAttempt(w, r, db, userid)
	if attempts == nil {
		fmt.Println("Gagal mendapatkan history failed login")
		return
	} else if len(attempts) > 2{
		
		return
	}
}

func CheckLoginAttempt(w http.ResponseWriter, r *http.Request, db *sql.DB, userid int) []FailedAttempt {
	rows, err := db.Query("select f.id, u.id, u.name, f.time from failed_log f JOIN users u ON f.userid = u.id WHERE userid=?", userid)
	if err != nil {
		log.Println(err)
		return nil
	}

	var attempt FailedAttempt
	var attempts []FailedAttempt

	for rows.Next() {
		if err := rows.Scan(&attempt.Id, &attempt.User.Id, &attempt.User.Name, &attempt.Time); err != nil {
			log.Println(err)
			return nil
		} else {
			attempts = append(attempts, attempt)
		}
	}
	return attempts
}

func AddFailedLogin(w http.ResponseWriter, r *http.Request, db *sql.DB, userid int) bool{
	_, errQuery := db.Exec("INSERT INTO failed_log(userid,time) values (?,CURRENT_TIMESTAMP)", userid)

	if errQuery == nil {
		return true
	} else {
		fmt.Println(errQuery)
		return false
	}
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
			sendResponse(w,400,"Something went wrong!!")
			return
		} else {
			break
		}
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
		FailedLogin(w, r, db, user.Id)
		sendResponse(w, 400, "Wrong Email/Password!!")
		return
	}else{
		DeleteFailedHistory(w, r, db, user.Id)
	}
	header := r.Header.Get("platform")

	sendResponse(w, 200, "Success login from "+header)
}

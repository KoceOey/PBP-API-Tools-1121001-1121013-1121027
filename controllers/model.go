package controllers

type User struct {
	Id       int    `json:"userid"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	State    int    `json:"state"`
}

type FailedAttempt struct {
	Id int `json:"userid"`
	User User `json:"user"`
	Time string `json:"time"` 
}

type ResponseData struct {
	Message string      `json:"message"`
	Status  int         `json:"status"`
	Data    interface{} `json:"data"`
}

type Response struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

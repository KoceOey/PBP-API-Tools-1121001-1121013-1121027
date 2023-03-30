package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cobaGomail/controllers"
	"github.com/go-co-op/gocron"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	s := gocron.NewScheduler(time.UTC)
	s.Every(5).Second().Do(controllers.FailedHistoryCheck)
	s.StartAsync()

	router.HandleFunc("/login", controllers.UserLogin).Methods("POST")

	http.Handle("/", router)
	fmt.Println("Connected to port 8080")
	log.Println("Connected to port 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

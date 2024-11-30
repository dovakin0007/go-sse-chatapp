package main

import (
	"fmt"
	"log"
	"net/http"

	"dovakin0007.com/chatapp/app"
)

func main() {
	http.HandleFunc("/login", app.LoginHandler)
	http.HandleFunc("/events", app.SseHandler)
	http.HandleFunc("/send", app.SendMessageHandler)
	http.HandleFunc("/rooms", app.ListRoomsHandler)
	http.HandleFunc("/private", app.SsePrivateRoomHandler)
	http.HandleFunc("/private/send", app.SendMessageToPrivateRoom)

	fmt.Println("Server is running on :8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}

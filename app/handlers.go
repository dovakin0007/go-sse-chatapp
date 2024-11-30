package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

var Users = struct {
	sync.RWMutex
	List map[string]bool
}{
	List: make(map[string]bool),
}

var roomManager = RoomManager{
	Rooms: make(map[string]*Room),
	Lock:  sync.RWMutex{},
}

func newRoom(name string) *Room {
	room := &Room{
		Name:         name,
		Clients:      make(map[chan []byte]string),
		Notifier:     make(chan []byte),
		AddClient:    make(chan ClientData),
		RemoveClient: make(chan ClientData),
		Done:         make(chan bool),
	}
	go room.Run()
	return room
}

func addUser(username string) {
	Users.Lock()
	defer Users.Unlock()
	Users.List[username] = true
}

func userExists(username string) bool {
	Users.RLock()
	defer Users.RUnlock()
	return Users.List[username]
}

func requireLogin(username string) error {
	if !userExists(username) {
		return fmt.Errorf("user %s is not logged in", username)
	}
	return nil
}

func getOrCreateRoom(name string) *Room {
	roomManager.Lock.Lock()
	defer roomManager.Lock.Unlock()
	if room, exists := roomManager.Rooms[name]; exists {
		return room
	}
	room := newRoom(name)
	roomManager.Rooms[name] = room
	return room
}

func CreatePrivateRoom(userA, userB string) string {
	if userA > userB {
		userA, userB = userB, userA
	}
	return fmt.Sprintf("%s-%s", userA, userB)
}

func getOrCreatePrivateRoom(userA, userB string) (*Room, error) {
	if !userExists(userA) || !userExists(userB) {
		return nil, fmt.Errorf("one or both users do not exist")
	}
	roomName := CreatePrivateRoom(userA, userB)
	roomManager.Lock.Lock()
	defer roomManager.Lock.Unlock()
	if room, exists := roomManager.Rooms[roomName]; exists {
		return room, nil
	}
	room := newRoom(roomName)
	roomManager.Rooms[roomName] = room
	return room, nil
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	setupHeaders(w)
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}
	Users.Lock()
	defer Users.Unlock()
	if _, exists := Users.List[username]; exists {
		w.Write([]byte("User already exists"))
		return
	}
	Users.List[username] = true
	w.Write([]byte("User created"))
}

func SseHandler(w http.ResponseWriter, r *http.Request) {
	setupHeaders(w)
	roomName := r.URL.Query().Get("room")
	username := r.URL.Query().Get("username")
	if roomName == "" || username == "" || requireLogin(username) != nil {
		http.Error(w, "Invalid parameters", http.StatusBadRequest)
		return
	}
	room := getOrCreateRoom(roomName)
	handleClientConnection(w, r, room, username)
}

func SsePrivateRoomHandler(w http.ResponseWriter, r *http.Request) {
	setupHeaders(w)
	userA := r.URL.Query().Get("userA")
	userB := r.URL.Query().Get("userB")
	username := r.URL.Query().Get("username")
	if username != userA && username != userB {
		http.Error(w, "Access denied", http.StatusUnauthorized)
		return
	}
	room, err := getOrCreatePrivateRoom(userA, userB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	handleClientConnection(w, r, room, username)
}

func SendMessageHandler(w http.ResponseWriter, r *http.Request) {
	setupHeaders(w)
	sendMessage(w, r, false)
}

func SendMessageToPrivateRoom(w http.ResponseWriter, r *http.Request) {
	setupHeaders(w)
	sendMessage(w, r, true)
}

func sendMessage(w http.ResponseWriter, r *http.Request, isPrivate bool) {
	setupHeaders(w)
	var roomName, username, message string
	username = r.FormValue("username")
	message = r.FormValue("message")
	if isPrivate {
		userA := r.FormValue("userA")
		userB := r.FormValue("userB")
		if username != userA && username != userB {
			http.Error(w, "Access denied", http.StatusUnauthorized)
			return
		}
		roomName = CreatePrivateRoom(userA, userB)
	} else {
		roomName = r.FormValue("room")
	}
	roomManager.Lock.RLock()
	defer roomManager.Lock.RUnlock()
	room, exists := roomManager.Rooms[roomName]
	if !exists {
		http.Error(w, "Room not found", http.StatusNotFound)
		return
	}
	jsonData, _ := json.Marshal(map[string]interface{}{
		"username": username,
		"message":  message,
	})
	room.Notifier <- jsonData
	w.Write([]byte("OK"))
}

func handleClientConnection(w http.ResponseWriter, r *http.Request, room *Room, username string) {
	setupHeaders(w)
	msgChan := make(chan []byte, 16)
	room.AddClient <- ClientData{Client: msgChan, Username: username}
	defer func() {
		room.RemoveClient <- ClientData{Client: msgChan, Username: username}
	}()
	for {
		select {
		case messageData := <-msgChan:
			fmt.Fprintf(w, "data: %s\n\n", string(messageData))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

func setupHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func ListRoomsHandler(w http.ResponseWriter, r *http.Request) {
	setupHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	roomManager.Lock.RLock()
	defer roomManager.Lock.RUnlock()
	roomNames := make([]string, 0, len(roomManager.Rooms))
	for roomName := range roomManager.Rooms {
		roomNames = append(roomNames, roomName)
	}
	json.NewEncoder(w).Encode(roomNames)
}

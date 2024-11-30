package app

import "sync"

var roomManager = RoomManager{
	Rooms: make(map[string]*Room),
	Lock:  sync.RWMutex{},
}

var userRoomType = make(map[string]string)

type Room struct {
	Name         string
	Clients      map[chan []byte]string
	Notifier     chan []byte
	AddClient    chan ClientData
	RemoveClient chan ClientData
	Done         chan bool
	AllowedUsers map[string]bool
}

type RoomManager struct {
	Rooms map[string]*Room
	Lock  sync.RWMutex
}

func newRoom(name string) *Room {
	room := &Room{
		Name:         name,
		Clients:      make(map[chan []byte]string),
		Notifier:     make(chan []byte),
		AddClient:    make(chan ClientData),
		RemoveClient: make(chan ClientData),
		Done:         make(chan bool),
		AllowedUsers: make(map[string]bool),
	}
	go room.Run()
	return room
}

func (r *Room) Run() {
	for {
		select {
		case clientData := <-r.AddClient:
			r.Clients[clientData.Client] = clientData.Username
		case clientData := <-r.RemoveClient:
			delete(r.Clients, clientData.Client)
			close(clientData.Client)
			if len(r.Clients) == 0 {
				roomManager.Lock.Lock()
				delete(roomManager.Rooms, r.Name)
				roomManager.Lock.Unlock()
				return
			}
		case message := <-r.Notifier:
			for clientChan := range r.Clients {
				clientChan <- message
			}
		}
	}
}

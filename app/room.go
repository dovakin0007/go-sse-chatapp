package app

import "sync"

type Room struct {
	Name         string
	Clients      map[chan []byte]string
	Notifier     chan []byte
	AddClient    chan ClientData
	RemoveClient chan ClientData
	Done         chan bool
}

type RoomManager struct {
	Rooms map[string]*Room
	Lock  sync.RWMutex
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

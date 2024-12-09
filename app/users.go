package app

import "sync"

var Users = struct {
	sync.RWMutex
	List map[string]bool
}{
	List: make(map[string]bool),
}

type Send struct {
	Username string `json:"username"`
	Message  string `json:"message"`
	Roomname string `json:"room"`
}

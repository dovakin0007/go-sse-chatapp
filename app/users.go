package app

import "sync"

var Users = struct {
	sync.RWMutex
	List map[string]bool
}{
	List: make(map[string]bool),
}

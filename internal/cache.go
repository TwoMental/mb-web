package internal

import (
	"sync"
	"time"
)

var (
	lock    sync.Mutex
	clients = make(map[string]*ModbusServer)
)

func DeleteConn(key string) {
	lock.Lock()
	defer lock.Unlock()
	if client, ok := clients[key]; ok {
		client.Close()
		delete(clients, key)
	}
}

func SaveConn(key string, client *ModbusServer) {
	lock.Lock()
	defer lock.Unlock()
	clients[key] = client
}

func GetConn(key string) (*ModbusServer, bool) {
	lock.Lock()
	defer lock.Unlock()
	client, ok := clients[key]
	return client, ok
}

// CleanConn closes connections that have been idle for too long.
func CleanConn() {
	lock.Lock()
	defer lock.Unlock()
	todo := []string{}
	for key, client := range clients {
		if time.Since(client.LastAlive) > 30*time.Minute {
			todo = append(todo, key)
			client.Close()
		}
	}
	for _, addr := range todo {
		delete(clients, addr)
	}
}

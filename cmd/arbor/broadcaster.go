package main

import (
	"log"
	"sync"

	"github.com/whereswaldon/arbor/lib/messages"
)

type Broadcaster struct {
	sync.RWMutex
	clients map[chan<- *messages.ArborMessage]struct{}
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[chan<- *messages.ArborMessage]struct{}),
	}
}

func (b *Broadcaster) Send(message *messages.ArborMessage) {
	b.RLock()
	for client := range b.clients {
		go func() {
			defer func() {
				if err := recover(); err != nil {
					log.Println("Error sending to client, removing: ", err)
					go b.remove(client)
				}
			}()
			client <- message
		}()
	}
	b.RUnlock()
}

func (b *Broadcaster) remove(client chan<- *messages.ArborMessage) {
	b.Lock()
	delete(b.clients, client)
	b.Unlock()
}

func (b *Broadcaster) Add(client chan<- *messages.ArborMessage) {
	b.Lock()
	b.clients[client] = struct{}{}
	b.Unlock()
}

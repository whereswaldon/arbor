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
	log.Println("Send acquired rlock")
	for client := range b.clients {
		go b.trySend(message, client)
	}
	b.RUnlock()
	log.Println("Send released rlock")
}

func (b *Broadcaster) trySend(message *messages.ArborMessage, client chan<- *messages.ArborMessage) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Error sending to client, removing: ", err)
			go b.remove(client)
		}
	}()
	log.Println("trying send to ", client)
	client <- message
}

func (b *Broadcaster) remove(client chan<- *messages.ArborMessage) {
	b.Lock()
	log.Println("removed acquired lock")
	delete(b.clients, client)
	b.Unlock()
	log.Println("remove released lock", client)
}

func (b *Broadcaster) Add(client chan<- *messages.ArborMessage) {
	b.Lock()
	log.Println("Add acquired lock")
	b.clients[client] = struct{}{}
	b.Unlock()
	log.Println("Add released lock")
}

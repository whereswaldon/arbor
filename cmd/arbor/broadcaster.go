package main

import (
	"log"

	"github.com/whereswaldon/arbor/lib/messages"
)

type Broadcaster struct {
	send       chan *messages.ArborMessage
	disconnect chan chan<- *messages.ArborMessage
	connect    chan chan<- *messages.ArborMessage
	clients    map[chan<- *messages.ArborMessage]struct{}
}

func NewBroadcaster() *Broadcaster {
	b := &Broadcaster{
		send:       make(chan *messages.ArborMessage),
		connect:    make(chan chan<- *messages.ArborMessage),
		disconnect: make(chan chan<- *messages.ArborMessage),
		clients:    make(map[chan<- *messages.ArborMessage]struct{}),
	}
	go b.dispatch()
	return b
}

func (b *Broadcaster) dispatch() {
	for {
		select {
		case message := <-b.send:
			for client := range b.clients {
				go b.trySend(message, client)
			}
		case newclient := <-b.connect:
			b.clients[newclient] = struct{}{}

		case deadclient := <-b.disconnect:
			delete(b.clients, deadclient)
		}
	}
}

func (b *Broadcaster) Send(message *messages.ArborMessage) {
	b.send <- message
}

func (b *Broadcaster) trySend(message *messages.ArborMessage, client chan<- *messages.ArborMessage) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Error sending to client, removing: ", err)
			b.disconnect <- client
		}
	}()
	log.Println("trying send to ", client)
	client <- message
}

func (b *Broadcaster) Add(client chan<- *messages.ArborMessage) {
	b.connect <- client
}

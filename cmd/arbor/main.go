package main

import (
	"log"
	"net"
	"os"

	. "github.com/whereswaldon/arbor/lib/messages"
)

func main() {
	messages := NewStore()
	broadcaster := NewBroadcaster()
	recents := NewRecents(10)
	address := ":7777"
	//serve
	if (len(os.Args) > 1) {
    		address = os.Args[1]
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Server listening on", address)
	m, err := NewMessage("Root message")
	err = m.AssignID()
	if err != nil {
		log.Println(err)
	}
	messages.Add(m)
	toWelcome := make(chan chan<- *ArborMessage)
	go handleWelcomes(m.UUID, recents, toWelcome)
	log.Println("Root message UUID is " + m.UUID)
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
		}
		fromClient := MakeMessageReader(conn)
		toClient := MakeMessageWriter(conn)
		broadcaster.Add(toClient)
		go handleClient(fromClient, toClient, recents, messages, broadcaster)
		toWelcome <- toClient
	}
}

func handleWelcomes(rootId string, recents *RecentList, toWelcome chan chan<- *ArborMessage) {
	for client := range toWelcome {
		msg := ArborMessage{
			Type:  WELCOME,
			Root:  rootId,
			Major: 0,
			Minor: 1,
		}
		msg.Recent = recents.Data()

		client <- &msg
		log.Println("Welcome message: ", msg.String())

	}
}

func handleClient(from <-chan *ArborMessage, to chan<- *ArborMessage, recents *RecentList, store *Store, broadcaster *Broadcaster) {
	for message := range from {
		switch message.Type {
		case QUERY:
			log.Println("Handling query for " + message.Message.UUID)
			go handleQuery(message, to, store)
		case NEW_MESSAGE:
			go handleNewMessage(message, recents, store, broadcaster)
		default:
			log.Println("Unrecognized message type", message.Type)
			continue
		}
	}
}

func handleQuery(msg *ArborMessage, out chan<- *ArborMessage, store *Store) {
	result := store.Get(msg.Message.UUID)
	if result == nil {
		log.Println("Unable to find queried id: " + msg.Message.UUID)
		return
	}
	msg.Message = result
	msg.Type = NEW_MESSAGE
	out <- msg
	log.Println("Query response: ", msg.String())
}

func handleNewMessage(msg *ArborMessage, recents *RecentList, store *Store, broadcaster *Broadcaster) {
	err := msg.Message.AssignID()
	if err != nil {
		log.Println("Error creating new message", err)
	}
	recents.Add(msg.Message.UUID)
	store.Add(msg.Message)
	broadcaster.Send(msg)
}

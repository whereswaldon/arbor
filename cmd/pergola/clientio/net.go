package clientio

import (
	"io"
	"log"

	messages "github.com/whereswaldon/arbor/lib/messages"
)

// HandleConn reads from the provided connection and writes new messages to the msgs
// channel as they come in.
func HandleNewMessages(conn io.ReadWriteCloser, msgs chan<- *messages.Message, welcomes chan<- *messages.ArborMessage) {
	readMessages := messages.MakeMessageReader(conn)
	defer close(msgs)
	for fromServer := range readMessages {
		switch fromServer.Type {
		case messages.WELCOME:
			welcomes <- fromServer
			close(welcomes)
			welcomes = nil
		case messages.NEW_MESSAGE:
			// add the new message
			msgs <- fromServer.Message
		default:
			log.Println("Unknown message type: ", fromServer.String)
			continue
		}
	}
}

// HandleRequests reads from the requestedIds and outbound channels and sends messages
// to the server. Any message id received on the requestedIds channel will be queried
// and any message received on the outbound channel will be sent as a new message
func HandleRequests(conn io.ReadWriteCloser, requestedIds <-chan string, outbund <-chan *messages.Message) {
	toServer := messages.MakeMessageWriter(conn)
	for {
		select {
		case queryId := <-requestedIds:
			a := &messages.ArborMessage{
				Type: messages.QUERY,
				Message: &messages.Message{
					UUID: queryId,
				},
			}
			toServer <- a
		case newMesg := <-outbund:
			a := &messages.ArborMessage{
				Type:    messages.NEW_MESSAGE,
				Message: newMesg,
			}
			toServer <- a
		}
	}
}

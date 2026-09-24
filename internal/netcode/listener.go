package netcode

import (
	"encoding/json"
	"io"
	"log"
	"net"

	"github.com/google/uuid"
	"github.com/mike-moseley/redoubt/internal/core"
	"github.com/mike-moseley/redoubt/internal/protocol"
)

func handleConn(conn net.Conn, inbound chan core.Event) {
	decoder := json.NewDecoder(conn)
	var clientEnv protocol.ClientEnvelope
	err := decoder.Decode(&clientEnv)
	if err != nil {
		log.Printf("Error decoding connect message: %v", err)
	}
	var connectMsg protocol.ConnectMessage
	err = json.Unmarshal(clientEnv.Payload, &connectMsg)
	if err != nil {
		log.Printf("Error unmarshaling connect message: %v", err)
	}

	// TODO: Handle auth
	session := core.Session{
		ID:       uuid.New(),
		Conn:     conn,
		Outbound: make(chan any, 100),
		Player: &core.Player{
			Name:          connectMsg.Name,
			Location:      &core.Position{X: 64, Y: 64},
			WorldLocation: &core.Position{X: 0, Y: 0},
			Vision:        3,
		},
		EntityID: 0,
	}
	inbound <- core.ConnectEvent{Session: &session}
	log.Printf("%s has connected", session.Player.Name)
	go func() {
		encoder := json.NewEncoder(conn)
		for msg := range session.Outbound {
			err := encoder.Encode(msg)
			if err != nil {
				log.Printf("Error serializing session data in listener.go: %v\n", err)
			}
		}
	}()
	for {
		err := decoder.Decode(&clientEnv)
		if err == io.EOF {
			inbound <- core.DisconnectEvent{SessionID: session.ID}
			log.Printf("%s has disconnected", session.Player.Name)
			return
		} else if err != nil {
			inbound <- core.DisconnectEvent{SessionID: session.ID}
			log.Printf("Connection error in listener.go: %v\n", err)
			return
		}
		inbound <- clientEnv
	}
}

func Listen(address string, inbound chan core.Event) {
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("Error starting listener in netcode/listener.go %v\n", err)
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("Error accepting connection in netcode/listener.go %v\n", err)
		}
		go handleConn(conn, inbound)
	}
}

package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"

	tea "charm.land/bubbletea/v2"
	"github.com/mike-moseley/redoubt/internal/bubble"
	"github.com/mike-moseley/redoubt/internal/protocol"
)

func main() {
	name := flag.String("name", "anon", "Player name")
	addr := flag.String("address", "localhost:9000", "Server Address")
	flag.Parse()
	conn, err := net.Dial("tcp", *addr)
	if err != nil {
		log.Printf("Error connecting to server: %v", err)
	}
	connect := protocol.ConnectMessage{
		Name: *name,
	}
	connectJson, err := json.Marshal(connect)
	if err != nil {
		log.Printf("Error marshaling connect: %v", err)
	}

	clientEnv := protocol.ClientEnvelope {
		Type: protocol.TypeConnect,
		Payload: connectJson,
	}

	encoder := json.NewEncoder(conn)
	err = encoder.Encode(clientEnv)
	if err != nil {
		log.Printf("Error encoding connect: %v", err)
	}

	model := bubble.NewModel(conn, encoder)
	tea.NewProgram(model).Run()
}

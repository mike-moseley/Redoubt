package main

import (
	"log"
	"net"

	tea "charm.land/bubbletea/v2"
	"github.com/mike-moseley/goAdvBuilder/internal/bubble"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:9000")
	if err != nil {
		log.Printf("Error connecting to server: %v", err)
	}
	model := bubble.NewModel(conn)
	tea.NewProgram(model).Run()
}

package core

import (
	"net"

	"github.com/google/uuid"
)

type Session struct {
	ID uuid.UUID
	Conn net.Conn
	Outbound chan any
	Player *Player
	EntityID uint64
}

type Player struct {
	Name string
	Location *Position
	WorldLocation *Position
	Vision uint8
}

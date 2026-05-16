package core

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Event interface {
	IsEvent()
}

type TickEvent struct {
}

func (t TickEvent) IsEvent() {
}

type ConnectEvent struct {
	Session *Session
}

func (c ConnectEvent) IsEvent() {
}

type DisconnectEvent struct {
	SessionID uuid.UUID
}

func (d DisconnectEvent) IsEvent() {

}

type CommandEnvelope struct {
	Type string
	Payload json.RawMessage
}

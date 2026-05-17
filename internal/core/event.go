package core

import (
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

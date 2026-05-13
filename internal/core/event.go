package core

import "github.com/google/uuid"

type Event interface {
	isEvent()
}

type TickEvent struct {
}

func (t TickEvent) isEvent() {
}

type ConnectEvent struct {
	Session *Session
}

func (c ConnectEvent) isEvent() {
}

type DisconnectEvent struct {
	SessionID uuid.UUID
}

func (d DisconnectEvent) isEvent() {

}

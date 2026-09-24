package protocol

import (
	"encoding/json"

	"github.com/mike-moseley/redoubt/internal/core"
)

type MoveMessage struct {
	Delta core.Position
}

type ClientEnvelope struct {
	Type    string
	Payload json.RawMessage
	Session *core.Session
}

type ConnectMessage struct {
	Name string
}

func (c ClientEnvelope) IsEvent() {}

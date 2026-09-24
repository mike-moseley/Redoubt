package protocol

import (
	"encoding/json"

	"github.com/mike-moseley/redoubt/internal/core"
)

type FOVSnapshot struct {
	Tiles     []core.TileType
	Width     uint8
	Height    uint8
	Entities  []core.RenderableEntity
	PlayerPos core.Position
}

type ServerEnvelope struct {
	Type    string
	Payload json.RawMessage
}

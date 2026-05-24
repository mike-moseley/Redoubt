package protocol

import (
	"encoding/json"

	"github.com/mike-moseley/goAdvBuilder/internal/core"
)

type FOVSnapshot struct {
	Tiles     []core.TileType
	Width     int
	Length    int
	Entities  []core.RenderableEntity
	PlayerPos core.Position
}

type ServerEnvelope struct {
	Type    string
	Payload json.RawMessage
}

package world

import (
	"github.com/mike-moseley/redoubt/internal/core"
)

type WorldCell struct {
	TypeID string
}

type WorldMap struct {
	Map []WorldCell
	Width uint8
	Height uint8
}

type LocalMap struct {
	Map []core.TileType
	Width uint8
	Height uint8
}

package world
import "github.com/mike-moseley/goAdvBuilder/internal/core"

type WorldCell struct {
	TypeID string
}

type WorldMap struct {
	Map []WorldCell
	Width uint8
}

type LocalMap struct {
	Map []core.TileType
	Width uint8
}

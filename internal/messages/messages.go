package messages


import "github.com/mike-moseley/goAdvBuilder/internal/core"

type FOVSnapshot struct {
	Tiles []core.TileType
	Width uint8
	Entities []core.RenderableEntity
	PlayerPos core.Position
}

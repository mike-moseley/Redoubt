package messages

import "github.com/mike-moseley/goAdvBuilder/internal/core"

type RenderableEntity struct {
	Position core.Position
	Render core.Renderable
}

type FOVSnapshot struct {
	Tiles []core.TileType
	Width uint8
	Entities []RenderableEntity
	PlayerPos core.Position
}

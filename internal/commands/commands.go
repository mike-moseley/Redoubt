package commands

import "github.com/mike-moseley/goAdvBuilder/internal/core"

type MoveEvent struct {
	Delta core.Position
	Session *core.Session
}
func (m MoveEvent) IsEvent() {}

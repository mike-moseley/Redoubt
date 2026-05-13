package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/mike-moseley/goAdvBuilder/internal/bubble"
)

func main() {
	model := bubble.NewModel()
	tea.NewProgram(model).Run()
}

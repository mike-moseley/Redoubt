package bubble

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mike-moseley/goAdvBuilder/internal/core"
	"github.com/mike-moseley/goAdvBuilder/internal/messages"
)

type TileSymbol struct {
	Symbol rune
	Style  lipgloss.Style
}

var Symbols = map[core.TileType]TileSymbol{
	core.Plain: {'.', lipgloss.NewStyle().Foreground(lipgloss.Color("#88ff88"))},
	core.Tree:  {'#', lipgloss.NewStyle().Foreground(lipgloss.Color("#22aa22"))},
}

var playerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))

type Model struct {
	Snapshot messages.FOVSnapshot
}

func NewModel() Model {
	blank := make([]core.TileType, 25)
	blank[15] = core.Tree
	return Model{
		Snapshot: messages.FOVSnapshot{
			Tiles:     blank,
			Width:     5,
			Entities:  []messages.RenderableEntity{},
			PlayerPos: core.Position{X: 2, Y: 2},
		},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	playerPos := &m.Snapshot.PlayerPos
	height := uint8(len(m.Snapshot.Tiles)) / m.Snapshot.Width
	switch msg := msg.(type) {

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if playerPos.CanMove(0, -1, m.Snapshot.Width, height) {
				playerPos.Y--
			}
		case "down", "j":
			if playerPos.CanMove(0, 1, m.Snapshot.Width, height) {
				playerPos.Y++
			}
		case "left", "h":
			if playerPos.CanMove(-1, 0, m.Snapshot.Width, height) {
				playerPos.X--
			}
		case "right", "l":
			if playerPos.CanMove(1, 0, m.Snapshot.Width, height) {
				playerPos.X++
			}
		case "enter", "space":
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	str := strings.Builder{}
	playerIdx := m.Snapshot.PlayerPos.ToIdx(int8(m.Snapshot.Width))
	entities := make(map[int]messages.RenderableEntity)

	for _,e := range m.Snapshot.Entities {
		idx := e.Position.ToIdx(int8(m.Snapshot.Width))
		entities[idx] = e
	}

	for i := range len(m.Snapshot.Tiles) {
		entity, ok := entities[i];
		if i == playerIdx {
			str.WriteString(playerStyle.Render("@"))
		} else if ok {
			// TODO: We will deal with lipgloss rendering when we have
			// ironed out entities implementation
			str.WriteRune(entity.Render.Symbol)
		} else {
			tile := Symbols[m.Snapshot.Tiles[i]]
			str.WriteString(tile.Style.Render(string(tile.Symbol)))
		}
		if (i+1)%int(m.Snapshot.Width) == 0 {
			str.WriteRune('\n')
		}
	}
	return tea.NewView(str.String())
}

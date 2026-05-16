package bubble

import (
	"encoding/json"
	"log"
	"net"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mike-moseley/goAdvBuilder/internal/commands"
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
var staleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4a0a6b"))

type Model struct {
	Snapshot messages.FOVSnapshot
	StaleMap []core.TileType
	Conn     net.Conn
}

func NewModel(conn net.Conn) Model {
	blank := make([]core.TileType, 256)
	blank[15] = core.Tree
	staleMap := make([]core.TileType, 256)
	return Model{
		Snapshot: messages.FOVSnapshot{
			Tiles:     blank,
			Width:     5,
			Entities:  []core.RenderableEntity{},
			PlayerPos: core.Position{X: 2, Y: 2},
		},
		StaleMap: staleMap,
		Conn:     conn,
	}
}

func (m Model) Init() tea.Cmd {
	return waitForSnapshot(m.Conn)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	playerPos := &m.Snapshot.PlayerPos
	height := uint8(len(m.Snapshot.Tiles)) / m.Snapshot.Width
	switch msg := msg.(type) {

	case messages.FOVSnapshot:
		m.Snapshot = msg
		for i, tile := range msg.Tiles {
			if tile != core.Unseen {
				m.StaleMap[i] = tile
			}
		}
		return m, waitForSnapshot(m.Conn)
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if playerPos.CanMove(0, -1, m.Snapshot.Width, height) {
				playerPos.Y--
			}
			delta := core.Position{X: 0, Y: -1}
			cmdEnv := prepareMoveCommand(delta)
			sendCommandEnvelope(m.Conn, cmdEnv)
			// log.Printf("\nYou moved up one tile\n")
		case "down", "j":
			if playerPos.CanMove(0, 1, m.Snapshot.Width, height) {
				playerPos.Y++
			}
			delta := core.Position{X: 0, Y: 1}
			cmdEnv := prepareMoveCommand(delta)
			sendCommandEnvelope(m.Conn, cmdEnv)
			// log.Printf("\nYou moved down one tile\n")
		case "left", "h":
			if playerPos.CanMove(-1, 0, m.Snapshot.Width, height) {
				playerPos.X--
			}
			delta := core.Position{X: -1, Y: 0}
			cmdEnv := prepareMoveCommand(delta)
			sendCommandEnvelope(m.Conn, cmdEnv)
			// log.Printf("\nYou moved left one tile\n")
		case "right", "l":
			if playerPos.CanMove(1, 0, m.Snapshot.Width, height) {
				playerPos.X++
			}
			delta := core.Position{X: 1, Y: 0}
			cmdEnv := prepareMoveCommand(delta)
			sendCommandEnvelope(m.Conn, cmdEnv)
			// log.Printf("\nYou moved right one tile\n")
		case "enter", "space":
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	str := strings.Builder{}
	playerIdx := m.Snapshot.PlayerPos.ToIdx(int8(m.Snapshot.Width))
	entities := make(map[int]core.RenderableEntity)

	for _, e := range m.Snapshot.Entities {
		idx := e.Position.ToIdx(int8(m.Snapshot.Width))
		entities[idx] = e
	}

	for i := range len(m.Snapshot.Tiles) {
		entity, ok := entities[i]
		entityStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(entity.Render.Color))
		if i == playerIdx {
			str.WriteString(playerStyle.Render("@"))
		} else if ok {
			str.WriteString(entityStyle.Render(string(entity.Render.Symbol)))
		} else {
			tile := Symbols[m.Snapshot.Tiles[i]]
			stale := Symbols[m.StaleMap[i]]
			if m.Snapshot.Tiles[i] == core.Unseen {
				if m.StaleMap[i] != core.Unseen {
					str.WriteString(staleStyle.Render(string(stale.Symbol)))
				}
			} else {
				str.WriteString(tile.Style.Render(string(tile.Symbol)))
			}
		}
		if (i+1)%int(m.Snapshot.Width) == 0 {
			str.WriteRune('\n')
		}
	}
	return tea.NewView(str.String())
}

func waitForSnapshot(conn net.Conn) tea.Cmd {
	return func() tea.Msg {
		var snapshot messages.FOVSnapshot
		err := json.NewDecoder(conn).Decode(&snapshot)
		if err != nil {
			log.Printf("Error decoding FOVSnapshot in bubble.go: %v", err)
		}
		return snapshot
	}
}

func sendCommandEnvelope(conn net.Conn, env core.CommandEnvelope) {
	go func() {
		encoder := json.NewEncoder(conn)
		err := encoder.Encode(env)
		if err != nil {
			log.Printf("Error encoding command: %v", err)
		}
	}()
}

func prepareMoveCommand(delta core.Position) core.CommandEnvelope {
	move := commands.MoveEvent{
		Delta: delta,
	}
	jsonMove, err := json.Marshal(move)
	if err != nil {
		log.Printf("Error marshaling move data: %v", err)
	}
	return core.CommandEnvelope{
		Type:    "move",
		Payload: jsonMove,
	}
}

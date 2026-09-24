package bubble

import (
	"encoding/json"
	"net"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/mike-moseley/redoubt/internal/core"
	"github.com/mike-moseley/redoubt/internal/protocol"
)

type TileSymbol struct {
	Symbol rune
	Style  lipgloss.Style
}

var Symbols = map[core.TileType]TileSymbol{
	core.Unseen: {'~', lipgloss.NewStyle().Foreground(lipgloss.Color("#ddddff"))},
	core.Plain:  {'.', lipgloss.NewStyle().Foreground(lipgloss.Color("#88ff88"))},
	core.Tree:   {'#', lipgloss.NewStyle().Foreground(lipgloss.Color("#22aa22"))},
}

var playerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
var staleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4a0a6b"))

type Model struct {
	Mode         GameMode
	Snapshot     protocol.FOVSnapshot
	StaleMap     []core.TileType
	Conn         net.Conn
	Encoder      *json.Encoder
	Decoder      *json.Decoder
	ChatBuffer   []string
	ChatInput    textinput.Model
	ChatViewport viewport.Model
	Height       int
	Width        int
}

func NewModel(conn net.Conn, encoder *json.Encoder) Model {
	blank := make([]core.TileType, 128*128)
	blank[15] = core.Tree
	staleMap := make([]core.TileType, 128*128)
	ti := textinput.New()
	ti.Placeholder = "Press space to chat"
	ti.CharLimit = 256
	return Model{
		Mode: ModeNormal,
		Snapshot: protocol.FOVSnapshot{
			Tiles:     blank,
			Width:     128,
			Height:    128,
			Entities:  []core.RenderableEntity{},
			PlayerPos: core.Position{X: 64, Y: 64},
		},
		StaleMap:     staleMap,
		Conn:         conn,
		Decoder:      json.NewDecoder(conn),
		Encoder:      encoder,
		ChatBuffer:   make([]string, 0, 128),
		ChatInput:    ti,
		ChatViewport: viewport.New(),
	}
}

func (m Model) Init() tea.Cmd {
	return waitForMessage(m.Decoder)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case protocol.FOVSnapshot:
		m.Snapshot = msg

		for i, tile := range msg.Tiles {
			if tile != core.Unseen {
				m.StaleMap[i] = tile
			}
		}
		return m, waitForMessage(m.Decoder)

	case protocol.Chat:
		str := strings.Builder{}
		name := msg.SourceName
		contents := msg.Contents
		str.WriteString(name)
		str.WriteString(" says '")
		str.WriteString(contents)
		str.WriteString("'")

		m.ChatBuffer = append(m.ChatBuffer, str.String())
		fullstr := strings.Join(m.ChatBuffer, "\n")
		m.ChatViewport.SetContent(fullstr)
		m.ChatViewport.GotoBottom()

		return m, waitForMessage(m.Decoder)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height - 2
		m.ChatInput.SetWidth(msg.Width)
		m.ChatViewport.SetWidth(msg.Width)
		m.ChatViewport.SetHeight((m.Height / 4))

	case tea.KeyPressMsg:
		switch m.Mode {
		case ModeNormal:
			return m.handleKeyNormal(msg)
		case ModeChat:
			return m.handleKeyChat(msg)
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	if m.Width == 0 {
		return tea.NewView("")
	}

	boxW := m.Width
	tileCols := boxW - paneSideW
	chatPaneH := paneTopH + m.ChatViewport.Height() + chatInputH + paneBottomH
	tileRows := m.Height - chatPaneH - paneTopH - paneBottomH

	entities := make(map[int]core.RenderableEntity)

	for _, e := range m.Snapshot.Entities {
		idx := e.Position.ToIdx(m.Snapshot.Width)
		entities[idx] = e
	}

	str := m.renderGameView(tileRows, tileCols, entities)

	activeColor := lipgloss.Color("#88ff88")
	inactiveColor := lipgloss.Color("#cccccc")

	gameColor := inactiveColor
	chatColor := inactiveColor

	if m.Mode == ModeNormal {
		gameColor = activeColor
	} else {
		chatColor = activeColor
	}

	gameView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(gameColor).
		Width(boxW).
		Height(tileRows + paneBottomH).
		Render(strings.TrimRight(str, "\n"))
	gmTitle := "─Surroundings"
	gmTopBorder := lipgloss.NewStyle().Foreground(gameColor).Render("╭" + gmTitle + strings.Repeat("─", boxW-len(gmTitle)) + "╮")
	gmPane := gmTopBorder + "\n" + gameView

	chat := m.ChatViewport.View()
	chat += "\n" + m.ChatInput.View()
	chatRendered := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(chatColor).
		Width(boxW).
		Height(m.ChatViewport.Height() + chatInputH + paneBottomH).
		Render(chat)
	chTitle := "─Chat"
	chTopBorder := lipgloss.NewStyle().Foreground(chatColor).Render("╭" + chTitle + strings.Repeat("─", boxW-len(chTitle)) + "╮")
	chPane := chTopBorder + "\n" + chatRendered
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, gmPane, chPane))
}

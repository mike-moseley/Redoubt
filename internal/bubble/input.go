package bubble

import (
	tea "charm.land/bubbletea/v2"
	"github.com/mike-moseley/redoubt/internal/core"
)

func (m Model) handleKeyNormal(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		delta := core.Position{X: 0, Y: -1}
		cmdEnv := prepareMoveMessage(delta)
		sendClientEnvelope(m.Encoder, cmdEnv)
		return m, nil
	case "down", "j":
		delta := core.Position{X: 0, Y: 1}
		cmdEnv := prepareMoveMessage(delta)
		sendClientEnvelope(m.Encoder, cmdEnv)
		return m, nil
	case "left", "h":
		delta := core.Position{X: -1, Y: 0}
		cmdEnv := prepareMoveMessage(delta)
		sendClientEnvelope(m.Encoder, cmdEnv)
		return m, nil
	case "right", "l":
		delta := core.Position{X: 1, Y: 0}
		cmdEnv := prepareMoveMessage(delta)
		sendClientEnvelope(m.Encoder, cmdEnv)
		return m, nil
	case "space", "'":
		m.Mode = ModeChat
		cmd := m.ChatInput.Focus()
		return m, cmd
	case "/":
		m.Mode = ModeChat
		cmd := m.ChatInput.Focus()
		m.ChatInput.SetValue("/")
		return m, cmd

	}
	return m, nil
}

func (m Model) handleKeyChat(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if msg.String() == "ctrl-c" {
		return m, tea.Quit
	}
	var tiCmd tea.Cmd
	var tivpCmd tea.Cmd
	m.ChatInput, tiCmd = m.ChatInput.Update(msg)
	switch msg.String() {
	case "esc":
		m.Mode = ModeNormal
		m.ChatInput.Blur()
		return m, tiCmd
	case "enter":
		text := m.ChatInput.Value()
		switch text {
		case "/exit":
			return m, tea.Quit
		}
		if text != "" {
			cmdEnv := prepareChatMessage(text)
			sendClientEnvelope(m.Encoder, cmdEnv)
		}
		m.ChatInput.Reset()
		m.Mode = ModeNormal
		m.ChatInput.Blur()
		return m, tiCmd
	case "up", "down":
		m.ChatViewport, tivpCmd = m.ChatViewport.Update(msg)
		return m, tivpCmd
	}
	return m, tiCmd
}

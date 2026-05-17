package bubble

import (
	"encoding/json"
	"log"

	tea "charm.land/bubbletea/v2"
	"github.com/mike-moseley/goAdvBuilder/internal/core"
	"github.com/mike-moseley/goAdvBuilder/internal/protocol"
)

func waitForMessage(decoder *json.Decoder) tea.Cmd {
	return func() tea.Msg {
		var env protocol.ServerEnvelope
		err := decoder.Decode(&env)
		if err != nil {
			log.Printf("Error decoding server envelope in bubble.go: %v", err)
		}
		switch env.Type {
		case protocol.TypeFOVSnapshot:
			var snapshot protocol.FOVSnapshot
			err := json.Unmarshal(env.Payload, &snapshot)
			if err != nil {
				log.Printf("Error unmarshaling FOV snapshot: %v", err)
			}
			return snapshot
		case protocol.TypeChatMessage:
			var msg protocol.Chat
			err := json.Unmarshal(env.Payload, &msg)
			if err != nil {
				log.Printf("Error unmarshaling chat message: %v", err)
			}
			return msg
		}
		return env
	}
}

func sendClientEnvelope(encoder *json.Encoder, env protocol.ClientEnvelope) {
	go func() {
		err := encoder.Encode(env)
		if err != nil {
			log.Printf("Error encoding command: %v", err)
		}
	}()
}

func prepareMoveMessage(delta core.Position) protocol.ClientEnvelope {
	move := protocol.MoveMessage{
		Delta: delta,
	}
	jsonMove, err := json.Marshal(move)
	if err != nil {
		log.Printf("Error marshaling move data: %v", err)
	}
	return protocol.ClientEnvelope{
		Type:    protocol.TypeMove,
		Payload: jsonMove,
	}
}

func prepareChatMessage(text string) protocol.ClientEnvelope {
	msg := protocol.Chat{
		Contents: text,
	}
	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error marshaling chat message")
	}
	return protocol.ClientEnvelope{
		Type:    protocol.TypeChatMessage,
		Payload: jsonMsg,
	}
}

package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mike-moseley/goAdvBuilder/internal/core"
	"github.com/mike-moseley/goAdvBuilder/internal/netcode"
	"github.com/mike-moseley/goAdvBuilder/internal/protocol"
	"github.com/mike-moseley/goAdvBuilder/internal/world"
)

func main() {
	store := core.NewStore()
	tiles := make([]core.TileType, 128*128)
	for i := range tiles {
		tiles[i] = core.Grass
	}
	chunks := make(map[core.Position]map[int]world.LocalMap, 128)
	globalMap := world.GlobalMap {
		Chunks: chunks,
	}
	localMap := make(map[int]world.LocalMap, 128)
	area := world.LocalMap{
		Map:    tiles,
		Width:  128,
		Length: 128,
	}
	localMap[0] = area
	globalMap.Chunks[core.Position{X: 0, Y: 0}] = localMap
	inbound := make(chan core.Event)
	pendingMoves := make(map[uuid.UUID]protocol.MoveMessage)
	log.Println("Server starting")
	go gameloop(store, globalMap, inbound, pendingMoves)
	netcode.Listen("0.0.0.0:9000", inbound)
}

func gameloop(store core.Store, globalMap world.GlobalMap, inbound chan core.Event, pendingMoves map[uuid.UUID]protocol.MoveMessage) {
	ticker := time.NewTicker(150 * time.Millisecond)
	go func() {
		for range ticker.C {
			inbound <- core.TickEvent{}
		}
	}()
	sessions := make(map[uuid.UUID]*core.Session)
	for {
		event := <-inbound
		switch e := event.(type) {
		case core.TickEvent:
			for _, session := range sessions {
				// playerPos := session.Player.Location
				player := session.Player
				playerChunk := globalMap.Chunks[*player.WorldLocation]
				playerLocalMap := playerChunk[player.Location.Z]

				moveEvent, ok := pendingMoves[session.ID]
				if ok {
					newPos := player.Location.AddPosition(moveEvent.Delta)
					session.Player.Location = &newPos
					store.LocalPosition[core.EntityID(session.EntityID)] = newPos
					delete(pendingMoves, session.ID)
				}
				tiles, renderableEntities := world.ComputeFOV(&store, &playerLocalMap, *player.Location, int(session.Player.Vision))
				snapshot := protocol.FOVSnapshot{
					Tiles:     tiles,
					Width:     playerLocalMap.Width,
					Length:    playerLocalMap.Length,
					Entities:  renderableEntities,
					PlayerPos: *player.Location,
				}
				servEnv := prepareFOVMessage(snapshot)
				session.Outbound <- servEnv
			}
		case core.ConnectEvent:
			e.Session.EntityID = uint64(store.NextID)
			store.NextID++
			uuid := e.Session.ID
			sessions[uuid] = e.Session
			eid := core.EntityID(e.Session.EntityID)
			store.LocalPosition[eid] = *e.Session.Player.Location
			store.Name[eid] = core.Name{Label: e.Session.Player.Name}
			playerRender := core.Renderable{
				ID:     e.Session.Player.Name,
				Symbol: '@',
				Color:  "#ffffff",
			}
			store.Renderable[eid] = playerRender
			store.WorldPosition[eid] = *e.Session.Player.WorldLocation
		case core.DisconnectEvent:
			session := sessions[e.SessionID]
			eid := core.EntityID(session.EntityID)
			close(session.Outbound)
			delete(sessions, e.SessionID)
			delete(store.LocalPosition, eid)
			delete(store.Name, eid)
			delete(store.Renderable, eid)
			delete(store.WorldPosition, eid)

		case protocol.ClientEnvelope:
			switch e.Type {
			case protocol.TypeMove:
				var move protocol.MoveMessage
				err := json.Unmarshal(e.Payload, &move)
				if err != nil {
					log.Printf("Error unmarshalling move data: %v", err)
				}
				if e.Session.Player.Location.CanMove(move.Delta) {
					pendingMoves[e.Session.ID] = move
				}
			case protocol.TypeChatMessage:
				var chat protocol.Chat
				err := json.Unmarshal(e.Payload, &chat)
				if err != nil {
					log.Printf("Error unmarshalling chat data: %v", err)
				}
				chat.Source = e.Session.ID
				chat.SourceName = e.Session.Player.Name
				// TODO: Expand destinations
				chat.Destination = protocol.DestinationLocal
				for _, session := range sessions {
					session.Outbound <- prepareChatMessage(chat)
				}
			}
		}
	}
}
func prepareFOVMessage(snapshot protocol.FOVSnapshot) protocol.ServerEnvelope {
	jsonSnapshot, err := json.Marshal(snapshot)
	if err != nil {
		log.Printf("Error marshaling snapshot data: %v", err)
	}
	return protocol.ServerEnvelope{
		Type:    protocol.TypeFOVSnapshot,
		Payload: jsonSnapshot,
	}
}
func prepareChatMessage(snapshot protocol.Chat) protocol.ServerEnvelope {
	jsonChat, err := json.Marshal(snapshot)
	if err != nil {
		log.Printf("Error marshaling chat data: %v", err)
	}
	return protocol.ServerEnvelope{
		Type:    protocol.TypeChatMessage,
		Payload: jsonChat,
	}
}

// func sendServerEnvelope(conn net.Conn, env protocol.ServerEnvelope) {
// 	go func() {
// 		encoder := json.NewEncoder(conn)
// 		err := encoder.Encode(env)
// 		if err != nil {
// 			log.Printf("Error encoding command: %v", err)
// 		}
// 	}()
// }

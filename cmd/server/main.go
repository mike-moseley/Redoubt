package main

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mike-moseley/goAdvBuilder/internal/commands"
	"github.com/mike-moseley/goAdvBuilder/internal/core"
	"github.com/mike-moseley/goAdvBuilder/internal/messages"
	"github.com/mike-moseley/goAdvBuilder/internal/netcode"
	"github.com/mike-moseley/goAdvBuilder/internal/world"
)

func main() {
	store := core.NewStore()
	tiles := make([]core.TileType, 256)
	for i := range tiles {
		tiles[i] = core.Plain
	}
	localMap := world.LocalMap{
		Map:   tiles,
		Width: 16,
	}
	inbound := make(chan core.Event)
	go gameloop(store, localMap, inbound)
	netcode.Listen("localhost:9000", inbound)
}

func gameloop(store core.Store, localMap world.LocalMap, inbound chan core.Event) {
	ticker := time.NewTicker(250 * time.Millisecond)
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
				tiles, renderableEntities := world.ComputeFOV(&store, &localMap, *session.Player.Location, int(session.Player.Vision))
				snapshot := messages.FOVSnapshot{
					Tiles:     tiles,
					Width:     localMap.Width,
					Entities:  renderableEntities,
					PlayerPos: *session.Player.Location,
				}
				session.Outbound <- snapshot
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
			
		case commands.MoveEvent:
			height := uint8(len(localMap.Map)/int(localMap.Width))
			newPos := e.Session.Player.Location.AddPosition(e.Delta)
			if (e.Session.Player.Location.CanMove(e.Delta.X, e.Delta.Y, localMap.Width, height)) {
				e.Session.Player.Location = &newPos
				log.Printf("%s has moved by %v", e.Session.Player.Name, e.Delta)
			}
		}
	}
}

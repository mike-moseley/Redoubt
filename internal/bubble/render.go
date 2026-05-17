package bubble

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mike-moseley/goAdvBuilder/internal/core"
)

func (m Model) renderGameView(gameviewHeight, gameviewWidth int, entities map[int]core.RenderableEntity) string {
	playerX := m.Snapshot.PlayerPos.X
	playerY := m.Snapshot.PlayerPos.Y
	str := strings.Builder{}
	for vy := range gameviewHeight + 2{
		for vx := range gameviewWidth {
			mapX := int(playerX) + (vx - gameviewWidth/2)
			mapY := int(playerY) + (vy - gameviewHeight/2)
			mapIdx := mapY*int(m.Snapshot.Width) + mapX
			entity, ok := entities[mapIdx]
			if mapX >= int(m.Snapshot.Width) || mapX < 0 {
				str.WriteString(staleStyle.Render("0"))
			} else if mapY >= int(m.Snapshot.Length) || mapY < 0 {
				str.WriteString(staleStyle.Render("0"))
			} else if mapX == int(playerX) && mapY == int(playerY) {
				str.WriteString(playerStyle.Render("@"))
			} else if ok {
				entityStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(entity.Render.Color))
				str.WriteString(entityStyle.Render(string(entity.Render.Symbol)))
			} else {
				tile := Symbols[m.Snapshot.Tiles[mapIdx]]
				stale := Symbols[m.StaleMap[mapIdx]]
				if m.Snapshot.Tiles[mapIdx] == core.Unseen {
					if m.StaleMap[mapIdx] != core.Unseen {
						str.WriteString(staleStyle.Render(string(stale.Symbol)))
					} else {
						str.WriteString(tile.Style.Render(string(tile.Symbol)))
					}
				} else {
					str.WriteString(tile.Style.Render(string(tile.Symbol)))
				}
			}
			if vx == gameviewWidth-1 {
				str.WriteRune('\n')
			}
		}
	}
	return str.String()
}

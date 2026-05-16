package world

import "github.com/mike-moseley/goAdvBuilder/internal/core"

func ComputeFOV(store *core.Store, localMap *LocalMap, playerPos core.Position, radius int) ([]core.TileType, []core.RenderableEntity) {
	var outputTiles []core.TileType
	var outputRE []core.RenderableEntity
	for idx, tile := range localMap.Map {
		tilePos := core.IdxToPosition(idx, localMap.Width)
		if playerPos.DistanceSquared(tilePos) <= radius*radius {
			outputTiles = append(outputTiles, tile)
		} else {
			outputTiles = append(outputTiles, core.Unseen)
		}
	}
	for _, renderableEntity := range core.Query(store.LocalPosition, store.Renderable) {
		if playerPos.DistanceSquared(renderableEntity.First) <= radius*radius {
			outputRE = append(outputRE, core.RenderableEntity{
				Position: renderableEntity.First,
				Render:   renderableEntity.Second,
			})
		}
	}
	return outputTiles, outputRE
}

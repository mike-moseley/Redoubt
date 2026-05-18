package world

import (
	"github.com/mike-moseley/goAdvBuilder/internal/core"
	"github.com/ojrac/opensimplex-go"
	"math"
)

type GlobalMap struct {
	Chunks map[core.Position]map[int]LocalMap
	ZMin      int
	ZMax      int
}

type LocalMap struct {
	Map []core.TileType
	// HeightMap is an array of Z-levels;
	// indices correspond to Map
	Width     uint8
	Length    uint8
}

func GenerateChunk(chunkPos core.Position, seed int64) (chunk map[int]LocalMap, zMin, zMax int) {
	noiseInstance := opensimplex.NewNormalized(seed)
	zMin = math.MaxInt
	zMax = math.MinInt

	chunk = make(map[int]LocalMap)

	for i := range int(core.LOCAL_MAP_LENGTH) * int(core.LOCAL_MAP_WIDTH) {
		localPos := core.IdxToPosition(i, core.LOCAL_MAP_WIDTH)
		worldX := float64(chunkPos.X*int(core.LOCAL_MAP_WIDTH)+localPos.X) / 600.0
		worldY := float64(chunkPos.Y*int(core.LOCAL_MAP_LENGTH)+localPos.Y) / 600.0
		tileZ := int(math.Floor(noiseInstance.Eval2(worldX, worldY) * 30))
		if tileZ > zMax {
			zMax = tileZ
		}
		if tileZ < zMin {
			zMin = tileZ
		}

		if _, ok := chunk[tileZ]; !ok {
			newTiles := make([]core.TileType, int(core.LOCAL_MAP_LENGTH)*int(core.LOCAL_MAP_WIDTH))
			chunk[tileZ] = LocalMap{
				Map: newTiles,
				Width: core.LOCAL_MAP_WIDTH,
				Length: core.LOCAL_MAP_LENGTH,
			}
		}
		lMap := chunk[tileZ]
		lMap.Map[i] = core.Grass
		chunk[tileZ] = lMap
	}

	return chunk, zMin, zMax
}

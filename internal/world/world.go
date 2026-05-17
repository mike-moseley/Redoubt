package world

import (
	"github.com/mike-moseley/goAdvBuilder/internal/core"
	"github.com/ojrac/opensimplex-go"
	"math"
)

type GlobalMap struct {
	Chunks map[core.Position]map[int]LocalMap
}

type LocalMap struct {
	Map []core.TileType
	// HeightMap is an array of Z-levels;
	// indices correspond to Map
	HeightMap []int
	Width     uint8
	Length    uint8
}

func GenerateChunk(chunkPos core.Position, seed int64) map[int]LocalMap {
	globalNoiseInstance := opensimplex.NewNormalized(seed)
	globalNoise := globalNoiseInstance.Eval2(float64(chunkPos.X), float64(chunkPos.Y))
	globalZ := int(math.Floor(globalNoise * 10))
	localSeed := seed + int64(chunkPos.X)*31 + int64(chunkPos.Y)*37
	localNoiseInstance := opensimplex.NewNormalized(localSeed)

	chunk := make(map[int]LocalMap)
	tiles := make([]core.TileType, int(core.LOCAL_MAP_LENGTH)*int(core.LOCAL_MAP_WIDTH))
	heightMap := make([]int, len(tiles))

	for i := range tiles {
		tiles[i] = core.Grass
		localPos := core.IdxToPosition(i, core.LOCAL_MAP_WIDTH)
		localNoise := localNoiseInstance.Eval2(float64(localPos.X), float64(localPos.Y))
		heightMap[i] = int(math.Floor(localNoise * 3))
	}

	localMap := LocalMap{
		Map:       tiles,
		HeightMap: heightMap,
		Width:     core.LOCAL_MAP_WIDTH,
		Length:    core.LOCAL_MAP_LENGTH,
	}

	chunk[globalZ] = localMap

	return chunk
}

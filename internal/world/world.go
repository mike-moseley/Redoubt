package world

import (
	"math"

	"github.com/mike-moseley/goAdvBuilder/internal/core"
	"github.com/ojrac/opensimplex-go"
)

type GlobalMap struct {
	Chunks    map[core.Position]map[int]LocalMap
	ZMin      int
	ZMax      int
	ZMaxIdx   int
	ZNoiseMap []float64
	ZMap      []int
}

type LocalMap struct {
	Map []core.TileType
	// HeightMap is an array of Z-levels;
	// indices correspond to Map
	Width  int
	Length int
}

func GenerateChunk(chunkPos, worldCenter core.Position, islandRadius int, seed int64) (chunk map[int]LocalMap, zMin, zMax, zMaxIdx int, rawZNoise []float64, zMap []int) {
	noiseInstance := opensimplex.NewNormalized(seed)
	zMin = math.MaxInt
	zMax = math.MinInt
	zMaxIdx = 0

	chunk = make(map[int]LocalMap)
	zMap = make([]int, 0, core.LOCAL_MAP_WIDTH*core.LOCAL_MAP_LENGTH)
	rawZNoise = make([]float64, core.LOCAL_MAP_LENGTH*core.LOCAL_MAP_WIDTH)

	for i := range int(core.LOCAL_MAP_LENGTH) * int(core.LOCAL_MAP_WIDTH) {
		localPos := core.IdxToPosition(i, core.LOCAL_MAP_WIDTH)
		worldX := float64(chunkPos.X*int(core.LOCAL_MAP_WIDTH) + localPos.X)
		worldY := float64(chunkPos.Y*int(core.LOCAL_MAP_LENGTH) + localPos.Y)
		worldPos := core.Position{X: int(worldX), Y: int(worldY)}
		distFromCenter := math.Sqrt(float64(worldPos.DistanceSquared(worldCenter)))
		distFromCenterNormed := distFromCenter / (float64(islandRadius))
		falloff := math.Pow(max(0.0, 1.0-distFromCenterNormed), 3.0)
		noise1 := noiseInstance.Eval2(worldX/600.0, worldY/600.0)
		noise2 := noiseInstance.Eval2(worldX/150.0, worldY/150.0)
		noise := noise1*0.7 + noise2*0.3
		combined := noise*0.4 + falloff*0.6
		rawZNoise[i] = combined

		tileZ := int(combined * 30)
		if tileZ > zMax {
			zMax = tileZ
			zMaxIdx = i
		}
		if tileZ < zMin {
			zMin = tileZ
		}

		zMap = append(zMap, tileZ)

		if _, ok := chunk[tileZ]; !ok {
			newTiles := make([]core.TileType, int(core.LOCAL_MAP_LENGTH)*int(core.LOCAL_MAP_WIDTH))
			chunk[tileZ] = LocalMap{
				Map:    newTiles,
				Width:  core.LOCAL_MAP_WIDTH,
				Length: core.LOCAL_MAP_LENGTH,
			}
		}
		lMap := chunk[tileZ]
		lMap.Map[i] = core.Grass
		chunk[tileZ] = lMap
	}

	return chunk, zMin, zMax, zMaxIdx, rawZNoise, zMap
}

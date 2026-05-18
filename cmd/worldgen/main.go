package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"

	"github.com/mike-moseley/goAdvBuilder/internal/core"
	"github.com/mike-moseley/goAdvBuilder/internal/world"
)

const (
	NUM_WORLDS = 2
	MAP_WIDTH  = 64
	MAP_LENGTH = 64
)

func main() {
	worlds := make([]world.GlobalMap, 0, NUM_WORLDS)

	globalPos := make([]core.Position, 0, MAP_WIDTH*MAP_LENGTH)
	for i := range MAP_LENGTH {
		for j := range MAP_WIDTH {
			pos := core.Position{X: i, Y: j}
			globalPos = append(globalPos, pos)
		}
	}

	for i := range NUM_WORLDS {
		worldZMin := math.MaxInt
		worldZMax := math.MinInt
		chunks := make(map[core.Position]map[int]world.LocalMap)
		for _, chunkPos := range globalPos {
			chunk, zMin, zMax := world.GenerateChunk(chunkPos, int64(i+1))
			chunks[chunkPos] = chunk
			if zMin < worldZMin {
				worldZMin = zMin
			}
			if zMax > worldZMax {
				worldZMax = zMax
			}
		}
		world := world.GlobalMap{Chunks: chunks, ZMin: worldZMin, ZMax: worldZMax}
		worlds = append(worlds, world)
	}

	for i, world := range worlds {
		img := image.NewRGBA(image.Rect(0, 0, int(core.LOCAL_MAP_WIDTH)*MAP_WIDTH, int(core.LOCAL_MAP_LENGTH)*MAP_LENGTH))
		for chunkPos, chunk := range world.Chunks {
			for z, lMap := range chunk {
				for i, tile := range lMap.Map {
					if tile == core.Unseen {
						continue
					}
					localPos := core.IdxToPosition(i, core.LOCAL_MAP_WIDTH)
					pixelX := chunkPos.X*int(core.LOCAL_MAP_WIDTH) + localPos.X
					pixelY := chunkPos.Y*int(core.LOCAL_MAP_LENGTH) + localPos.Y
					brightness := uint8((z - world.ZMin) * 255 / (world.ZMax - world.ZMin))
					img.SetRGBA(pixelX, pixelY, color.RGBA{brightness, brightness, brightness, 255})
				}
			}
		}
		// This `i` corresponds to the `i` in `i := range NUM_WORLDS`
		filename := fmt.Sprintf("seed-%d-heighmap.png", i)
		f, _ := os.Create(filename)
		defer f.Close()
		png.Encode(f, img)
	}
}


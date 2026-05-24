package main

import (
	"container/heap"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"

	"github.com/mike-moseley/goAdvBuilder/internal/core"
	"github.com/mike-moseley/goAdvBuilder/internal/world"
)

type heapTile struct {
	idx int
	z   float64
}

type TileHeap []heapTile

func (h TileHeap) Len() int           { return len(h) }
func (h TileHeap) Less(i, j int) bool { return h[i].z < h[j].z }
func (h TileHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *TileHeap) Push(x any)        { *h = append(*h, x.(heapTile)) }
func (h *TileHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

const (
	NUM_WORLDS              = 1
	SEED                    = 7
	TOTAL_LENGTH            = core.MAP_LENGTH * int(core.LOCAL_MAP_LENGTH)
	TOTAL_WIDTH             = core.MAP_WIDTH * int(core.LOCAL_MAP_WIDTH)
	ISLAND_RADIUS           = 5000
	SEALEVEL                = 10
	WATER_SOURCE_MIN_HEIGHT = 12
	NUM_WATER_SOURCES       = 1000
	RIVER_FLUX_THRESHOLD    = 3
	RIVER_RADIUS            = 3
)

func main() {
	worlds := make([]world.GlobalMap, 0, NUM_WORLDS)

	globalPos := make([]core.Position, 0, core.MAP_WIDTH*core.MAP_LENGTH)
	worldZMap := make([]int, TOTAL_LENGTH*TOTAL_WIDTH)
	mapCenter := core.Position{X: TOTAL_WIDTH / 2, Y: TOTAL_LENGTH / 2}
	oceanMap := make([]bool, TOTAL_WIDTH*TOTAL_LENGTH)
	lakeMap := make([]bool, TOTAL_WIDTH*TOTAL_LENGTH)
	flowMap := make([]int, TOTAL_LENGTH*TOTAL_WIDTH)
	fluxMap := make([]int, TOTAL_LENGTH*TOTAL_WIDTH)
	waterSourceMap := []int{}

	for i := range core.MAP_LENGTH {
		for j := range core.MAP_WIDTH {
			pos := core.Position{X: i, Y: j}
			globalPos = append(globalPos, pos)
		}
	}

	for i := range NUM_WORLDS {
		w := generateWorld(globalPos, mapCenter, worldZMap, &waterSourceMap, SEED+i)
		worlds = append(worlds, w)
	}

	calcOcean(worlds[0], oceanMap)
	fillSinks(worlds[0], oceanMap, lakeMap)
	genFlowmap(worlds[0], mapCenter, oceanMap, flowMap)
	genRivers(worlds[0], oceanMap, flowMap, waterSourceMap, fluxMap)
	log.Printf("zmin: %d  zmax: %d", worlds[0].ZMin, worlds[0].ZMax)
	log.Printf("water sources: %d", len(waterSourceMap))
	// after genRivers:
	count := 0
	for _, f := range fluxMap {
		if f > 0 {
			count++
		}
	}
	log.Printf("flux tiles: %d", count)

	os.MkdirAll("images", 0755)
	riverMap := dilateFlux(fluxMap)
	for i, w := range worlds {
		img := drawMap(w, oceanMap, lakeMap, riverMap)
		// This `i` corresponds to the `i` in `i := range NUM_WORLDS`
		filename := fmt.Sprintf("images/seed-%d-heightmap.png", i)
		f, _ := os.Create(filename)
		png.Encode(f, img)
		f.Close()
	}
}

func generateBoundarySlice() []int {
	boundaries := []int{}

	for x := range TOTAL_WIDTH {
		boundaries = append(boundaries, 0*TOTAL_WIDTH+x)
		boundaries = append(boundaries, (TOTAL_LENGTH-1)*TOTAL_WIDTH+x)
	}

	for y := range TOTAL_LENGTH {
		boundaries = append(boundaries, y*TOTAL_WIDTH+0)
		boundaries = append(boundaries, y*TOTAL_WIDTH+(TOTAL_WIDTH-1))
	}

	return boundaries
}

func calcOcean(w world.GlobalMap, oceanMap []bool) {
	boundaries := generateBoundarySlice()
	startTilesIdx := []int{}
	for _, idx := range boundaries {
		if w.ZMap[idx] < SEALEVEL {
			oceanMap[idx] = true
			startTilesIdx = append(startTilesIdx, idx)
		}
	}
	head := 0
	for head < len(startTilesIdx) {
		idx := startTilesIdx[head]
		head++
		pos := core.IdxToPosition(idx, TOTAL_WIDTH)

		northIdx := pos.AddPosition(core.North).ToIdx(TOTAL_WIDTH)
		southIdx := pos.AddPosition(core.South).ToIdx(TOTAL_WIDTH)
		eastIdx := pos.AddPosition(core.East).ToIdx(TOTAL_WIDTH)
		westIdx := pos.AddPosition(core.West).ToIdx(TOTAL_WIDTH)

		if pos.CanMoveBounded(core.North, TOTAL_WIDTH, TOTAL_LENGTH) && w.ZMap[northIdx] < SEALEVEL && !oceanMap[northIdx] {
			oceanMap[northIdx] = true
			startTilesIdx = append(startTilesIdx, northIdx)
		}
		if pos.CanMoveBounded(core.South, TOTAL_WIDTH, TOTAL_LENGTH) && w.ZMap[southIdx] < SEALEVEL && !oceanMap[southIdx] {
			oceanMap[southIdx] = true
			startTilesIdx = append(startTilesIdx, southIdx)
		}
		if pos.CanMoveBounded(core.East, TOTAL_WIDTH, TOTAL_LENGTH) && w.ZMap[eastIdx] < SEALEVEL && !oceanMap[eastIdx] {
			oceanMap[eastIdx] = true
			startTilesIdx = append(startTilesIdx, eastIdx)
		}
		if pos.CanMoveBounded(core.West, TOTAL_WIDTH, TOTAL_LENGTH) && w.ZMap[westIdx] < SEALEVEL && !oceanMap[westIdx] {
			oceanMap[westIdx] = true
			startTilesIdx = append(startTilesIdx, westIdx)
		}
	}
}

func fillSinks(w world.GlobalMap, oceanMap []bool, lakeMap []bool) {
	resolved := make([]bool, len(w.ZNoiseMap))
	h := &TileHeap{}
	heap.Init(h)

	for i, isOcean := range oceanMap {
		if isOcean {
			heap.Push(h, heapTile{idx: i, z: w.ZNoiseMap[i]})
			resolved[i] = true
		}
	}

	for h.Len() > 0 {
		curr := heap.Pop(h).(heapTile)
		pos := core.IdxToPosition(curr.idx, TOTAL_WIDTH)

		for _, dir := range []core.Position{core.North, core.South, core.East, core.West} {
			if !pos.CanMoveBounded(dir, TOTAL_WIDTH, TOTAL_LENGTH) {
				continue
			}
			neighborIdx := pos.AddPosition(dir).ToIdx(TOTAL_WIDTH)
			if resolved[neighborIdx] {
				continue
			}
			resolved[neighborIdx] = true
			if w.ZNoiseMap[neighborIdx] < curr.z {
				w.ZNoiseMap[neighborIdx] = curr.z + 1e-10
				lakeMap[neighborIdx] = true
			}
			heap.Push(h, heapTile{idx: neighborIdx, z: w.ZNoiseMap[neighborIdx]})
		}
	}
}

func genFlowmap(w world.GlobalMap, mapCenter core.Position, oceanMap []bool, flowMap []int) {
	tilesIdx := []int{}
	for i, oceanTile := range oceanMap {
		if !oceanTile {
			tilesIdx = append(tilesIdx, i)
		}
	}

	// slices.SortFunc(tilesIdx, func(a, b int) int { return w.ZMap[b] - w.ZMap[a] })

	head := 0
	for head < len(tilesIdx) {
		idx := tilesIdx[head]
		head++
		pos := core.IdxToPosition(idx, TOTAL_WIDTH)

		northIdx := -1
		southIdx := -1
		eastIdx := -1
		westIdx := -1

		if pos.CanMoveBounded(core.North, TOTAL_WIDTH, TOTAL_LENGTH) {
			northIdx = pos.AddPosition(core.North).ToIdx(TOTAL_WIDTH)
		}

		if pos.CanMoveBounded(core.South, TOTAL_WIDTH, TOTAL_LENGTH) {
			southIdx = pos.AddPosition(core.South).ToIdx(TOTAL_WIDTH)
		}

		if pos.CanMoveBounded(core.East, TOTAL_WIDTH, TOTAL_LENGTH) {
			eastIdx = pos.AddPosition(core.East).ToIdx(TOTAL_WIDTH)
		}

		if pos.CanMoveBounded(core.West, TOTAL_WIDTH, TOTAL_LENGTH) {
			westIdx = pos.AddPosition(core.West).ToIdx(TOTAL_WIDTH)
		}

		minIdx := idx
		minZ := w.ZNoiseMap[idx]

		for _, neighborIdx := range []int{northIdx, southIdx, eastIdx, westIdx} {
			if neighborIdx != -1 && w.ZNoiseMap[neighborIdx] < minZ {
				minZ = w.ZNoiseMap[neighborIdx]
				minIdx = neighborIdx
			}
		}

		flowMap[idx] = minIdx
	}
}

func genRivers(w world.GlobalMap, oceanMap []bool, flowMap []int, waterSourceMap []int, fluxMap []int) {
	sources := []int{}
	rng := rand.New(rand.NewSource(int64(SEED)))
	if len(waterSourceMap) == 0 {
		sources = append(sources, w.ZMaxIdx)
	} else {
		for range NUM_WATER_SOURCES {
			sources = append(sources, waterSourceMap[rng.Intn(len(waterSourceMap))])
		}
	}
	for _, sourceIdx := range sources {
		curr := sourceIdx
		// checkedIdx := map[int]int{}
		for !oceanMap[curr] && flowMap[curr] != curr {
			// _, ok := checkedIdx[sourceIdx]
			// if ok {
			// 	break
			// }
			// checkedIdx[curr] = curr
			fluxMap[curr]++
			curr = flowMap[curr]
		}
	}
}

func dilateFlux(fluxMap []int) []bool {
	riverMap := make([]bool, len(fluxMap))
	for idx, f := range fluxMap {
		if f < RIVER_FLUX_THRESHOLD {
			continue
		}
		pos := core.IdxToPosition(idx, TOTAL_WIDTH)
		for dy := -RIVER_RADIUS; dy <= RIVER_RADIUS; dy++ {
			for dx := -RIVER_RADIUS; dx <= RIVER_RADIUS; dx++ {
				nx, ny := pos.X+dx, pos.Y+dy
				if nx >= 0 && nx < TOTAL_WIDTH && ny >= 0 && ny < TOTAL_LENGTH {
					riverMap[ny*TOTAL_WIDTH+nx] = true
				}
			}
		}
	}
	return riverMap
}

func drawMap(w world.GlobalMap, oceanMap []bool, lakeMap []bool, riverMap []bool) image.Image {
	var wg sync.WaitGroup
	var mu sync.Mutex
	img := image.NewRGBA(image.Rect(0, 0, int(core.LOCAL_MAP_WIDTH)*core.MAP_WIDTH, int(core.LOCAL_MAP_LENGTH)*core.MAP_LENGTH))
	for chunkPos, chunk := range w.Chunks {
		wg.Add(1)
		go func(cp core.Position, c map[int]world.LocalMap, world world.GlobalMap) {
			defer wg.Done()

			for z, lMap := range chunk {
				for i, tile := range lMap.Map {
					if tile == core.Unseen {
						continue
					}
					localPos := core.IdxToPosition(i, core.LOCAL_MAP_WIDTH)
					pixelX := chunkPos.X*int(core.LOCAL_MAP_WIDTH) + localPos.X
					pixelY := chunkPos.Y*int(core.LOCAL_MAP_LENGTH) + localPos.Y
					worldIdx := pixelY*TOTAL_WIDTH + pixelX
					var brightness uint8
					if oceanMap[worldIdx] {
						mu.Lock()
						img.SetRGBA(pixelX, pixelY, color.RGBA{0, 0, 255, 255})
						mu.Unlock()
					} else if lakeMap[worldIdx] {
						mu.Lock()
						img.SetRGBA(pixelX, pixelY, color.RGBA{175, 175, 255, 255})
						mu.Unlock()
					} else if riverMap[worldIdx] {
						mu.Lock()
						img.SetRGBA(pixelX, pixelY, color.RGBA{0, 100, 255, 255})
						mu.Unlock()
					} else {
						brightness = uint8((z - world.ZMin) * 255 / (world.ZMax - world.ZMin))
						mu.Lock()
						img.SetRGBA(pixelX, pixelY, color.RGBA{brightness, brightness, brightness, 255})
						mu.Unlock()
					}
				}
			}
		}(chunkPos, chunk, w)
	}
	wg.Wait()
	return img
}

func generateWorld(globalPos []core.Position, mapCenter core.Position, worldZMap []int, waterSourceMap *[]int, seed int) world.GlobalMap {
	worldZMin := math.MaxInt
	worldZMax := math.MinInt
	worldZMaxIdx := -1
	worldZNoiseMap := make([]float64, TOTAL_LENGTH*TOTAL_WIDTH)
	var wg sync.WaitGroup
	var mu sync.Mutex
	chunks := make(map[core.Position]map[int]world.LocalMap, len(globalPos))
	chunksZmaps := make(map[core.Position][]int, len(globalPos))
	chunksRawZmaps := make(map[core.Position][]float64, len(globalPos))

	for _, chunkPos := range globalPos {
		wg.Add(1)
		go func(pos core.Position) {
			defer wg.Done()
			chunk, zMin, zMax, zMaxIdx, rawZNoise, zMap := world.GenerateChunk(pos, mapCenter, ISLAND_RADIUS, int64(seed))
			mu.Lock()
			chunks[pos] = chunk
			chunksZmaps[pos] = zMap
			chunksRawZmaps[pos] = rawZNoise
			if zMin < worldZMin {
				worldZMin = zMin
			}
			if zMax > worldZMax {
				worldZMax = zMax
				worldZMaxIdx = zMaxIdx
			}
			mu.Unlock()
		}(chunkPos)
	}
	wg.Wait()

	var zmapWg sync.WaitGroup
	for chunkPos, zMap := range chunksZmaps {
		zmapWg.Add(1)
		go func(pos core.Position, zm []int) {
			defer zmapWg.Done()
			for i, z := range zm {
				localPos := core.IdxToPosition(i, core.LOCAL_MAP_WIDTH)
				worldX := pos.X*int(core.LOCAL_MAP_WIDTH) + localPos.X
				worldY := pos.Y*int(core.LOCAL_MAP_LENGTH) + localPos.Y
				worldIdx := worldY*TOTAL_WIDTH + worldX
				worldZMap[worldIdx] = z
				if z > WATER_SOURCE_MIN_HEIGHT {
					mu.Lock()
					*waterSourceMap = append(*waterSourceMap, worldIdx)
					mu.Unlock()
				}
			}
		}(chunkPos, zMap)
	}
	zmapWg.Wait()

	var rawZmapWg sync.WaitGroup
	for chunkPos, zMap := range chunksRawZmaps {
		rawZmapWg.Add(1)
		go func(pos core.Position, zm []float64) {
			defer rawZmapWg.Done()
			for i, z := range zm {
				localPos := core.IdxToPosition(i, core.LOCAL_MAP_WIDTH)
				worldX := pos.X*int(core.LOCAL_MAP_WIDTH) + localPos.X
				worldY := pos.Y*int(core.LOCAL_MAP_LENGTH) + localPos.Y
				worldIdx := worldY*TOTAL_WIDTH + worldX
				worldZNoiseMap[worldIdx] = z
			}
		}(chunkPos, zMap)
	}
	rawZmapWg.Wait()
	return world.GlobalMap{Chunks: chunks, ZMin: worldZMin, ZMax: worldZMax, ZMaxIdx: worldZMaxIdx, ZNoiseMap: worldZNoiseMap, ZMap: worldZMap}
}

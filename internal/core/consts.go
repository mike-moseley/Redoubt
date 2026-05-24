package core

const (
	LOCAL_MAP_WIDTH  = 128
	LOCAL_MAP_LENGTH = 128
	MAP_WIDTH        = 64
	MAP_LENGTH       = 64
)

var (
	North = Position{X: 0, Y: -1}
	South = Position{X: 0, Y: 1}
	East = Position{X: 1, Y: 0}
	West = Position{X: -1, Y: 0}
	NorthEast = Position{X: 1, Y: -1}
	NorthWest = Position{X: -1, Y: -1}
	SouthEast = Position{X: 1, Y: 1}
	SouthWest = Position{X: -1, Y: 1}
)

package core

type Position struct {
	X int8
	Y int8
}

func (p Position) ToIdx(width int8) int {
	return int(p.X + p.Y*width)
}

func (p Position) CanMove(dx, dy int8, width uint8, height uint8) bool {
	// Cast up to int16 for comparison
	newX := int16(p.X) + int16(dx)
	newY := int16(p.Y) + int16(dy)

	if (newX >= int16(width)) || (newX < 0) {
		return false
	}

	if (newY >= int16(height)) || (newY < 0) {
		return false
	}

	return true
}

func (p Position) DistanceSquared(other Position) int {
	dx := int(p.X - other.X)
	dy := int(p.Y - other.Y)
	return dx*dx + dy*dy
}

func (p Position) AddPosition(other Position) Position {
	return Position{
		X:	p.X + other.X,
		Y:	p.Y + other.Y,
	}
}

func IdxToPosition(idx int, width uint8) Position {
	widthInt := int(width)
	y := idx / widthInt
	x := idx % widthInt
	return Position{
		X: int8(x),
		Y: int8(y),
	}
}

type Renderable struct {
	ID     string
	Symbol rune
	Color  string // RGB hex string
}

type Name struct {
	Label string
}

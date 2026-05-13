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

type Renderable struct {
	ID     string
	Symbol rune
	Color  string // RGB hex string
}

type Name struct {
	Label string
}

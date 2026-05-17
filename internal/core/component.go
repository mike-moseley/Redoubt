package core

type Position struct {
	X int
	Y int
	Z int
}

func (p Position) ToIdx(width uint8) int {
	x := p.X
	y := p.Y
	w := int(width)
	return x + y*w
}

func (p Position) CanMove(delta Position) bool {
	newX := p.X + delta.X
	newY := p.Y + delta.Y

	if (newX >= int(LOCAL_MAP_WIDTH)) || (newX < 0) {
		return false
	}

	if (newY >= int(LOCAL_MAP_LENGTH)) || (newY < 0) {
		return false
	}

	return true
}

func (p Position) DistanceSquared(other Position) int {
	dx := p.X - other.X
	dy := p.Y - other.Y
	return dx*dx + dy*dy
}

func (p Position) DistanceSquared3D(other Position) int {
	dx := p.X - other.X
	dy := p.Y - other.Y
	dz := p.Z - other.Z
	return dx*dx + dy*dy + dz*dz
}

func (p Position) AddPosition(other Position) Position {
	return Position{
		X: p.X + other.X,
		Y: p.Y + other.Y,
		Z: p.Z + other.Z,
	}
}

func IdxToPosition(idx int, width uint8) Position {
	widthInt := int(width)
	y := idx / widthInt
	x := idx % widthInt
	return Position{
		X: x,
		Y: y,
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

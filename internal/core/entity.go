package core

type EntityID uint64

type Store struct {
	LocalPosition   map[EntityID]Position
	WorldPosition map[EntityID]Position
	Renderable map[EntityID]Renderable
	Name       map[EntityID]Name
	NextID     EntityID
}

func NewStore() Store {
	return Store {
		LocalPosition: make(map[EntityID]Position),
		WorldPosition: make(map[EntityID]Position),
		Renderable: make(map[EntityID]Renderable),
		Name: make(map[EntityID]Name),
	}
}

type RenderableEntity struct {
	Position Position
	Render Renderable
}

type Pair[A, B any] struct {
	First A
	Second B
}

func Query[A, B any](mapA map[EntityID]A, mapB map[EntityID]B) func(yield func(EntityID, Pair[A, B]) bool) {
	return func(yield func(EntityID, Pair[A, B]) bool) {
		if len(mapA) >= len(mapB) {
			for id, a := range mapA {
				b, ok := mapB[id]
				if ok && !yield(id, Pair[A, B]{First: a, Second: b}) {
					return
				}
			}
		} else {
			for id, b := range mapB {
				a, ok := mapA[id]
				if ok && !yield(id, Pair[A, B]{First: a, Second: b}) {
					return
				}
			}
		}
	}
}

package core

type EntityID uint64

type Store struct {
	LocalPosition   map[EntityID]Position
	WorldPosition map[EntityID]Position
	Renderable map[EntityID]Renderable
	Name       map[EntityID]Name
	NextID     EntityID
}

func Query[A, B any](mapA map[EntityID]A, mapB map[EntityID]B) func(yield func(EntityID, A, B) bool) {
	return func(yield func(EntityID, A, B) bool) {
		if len(mapA) >= len(mapB) {
			for id, a := range mapA {
				b, ok := mapB[id]
				if ok && !yield(id, a, b) {
					return
				}
			}
		} else {
			for id, b := range mapB {
				a, ok := mapA[id]
				if ok && !yield(id, a, b) {
					return
				}
			}
		}
	}
}

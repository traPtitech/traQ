package set

import "iter"

type Set[T comparable] struct {
	set map[T]struct{}
}

func New[T comparable]() Set[T] {
	return Set[T]{
		set: make(map[T]struct{}),
	}
}

func (set *Set[T]) Add(v ...T) {
	for _, v := range v {
		set.set[v] = struct{}{}
	}
}

func (set *Set[T]) Remove(v ...T) {
	for _, v := range v {
		delete(set.set, v)
	}
}

func (set *Set[T]) Contains(v T) bool {
	_, ok := set.set[v]
	return ok
}

func (set *Set[T]) Len() int {
	return len(set.set)
}

func (set *Set[T]) Values() iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range set.set {
			if !yield(v) {
				return
			}
		}
	}
}

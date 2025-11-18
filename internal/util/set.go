package util

type Set[T comparable] struct {
	m map[T]struct{}
}

func (s *Set[T]) Add(v T) {
	s.m[v] = struct{}{}
}

func (s *Set[T]) Has(v T) bool {
	_, ok := s.m[v]
	return ok
}

func (s *Set[T]) Del(v T) {
	delete(s.m, v)
}

func (s *Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s.m))
	for key := range s.m {
		result = append(result, key)
	}
	return result
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{m: make(map[T]struct{})}
}

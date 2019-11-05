package set

import "reflect"

type Set struct {
	values map[interface{}]bool
}

func New() *Set {
	return &Set{
		values: make(map[interface{}]bool),
	}
}

func (s *Set) Put(v interface{}) (has bool) {
	has = s.Has(v)
	s.values[v] = true
	return
}

func (s *Set) Has(v interface{}) bool {
	return s.values[v]
}

func (s *Set) Remove(v interface{}) {
	delete(s.values, v)
}

func (s *Set) Clear() {
	s.values = make(map[interface{}]bool)
}

func (s *Set) Len() int {
	return len(s.values)
}

func (s *Set) Array() []interface{} {
	val := reflect.ValueOf(s.values)
	valKeys:= val.MapKeys()
	values := make([]interface{}, len(valKeys))
	for i, k := range valKeys {
		values[i] = k.Interface()
	}
	return values
}

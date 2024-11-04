package tdag

import (
	"fmt"
	"sync"
)

type TStore struct {
	items map[string]interface{}
	mu    sync.Mutex
}

func (s *TStore) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = value
}

func (s *TStore) Get(key string) (interface{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.items[key]
	if !ok {
		return nil, fmt.Errorf("key %s not found", key)
	}
	return value, nil
}

func NewStore() *TStore {
	return &TStore{
		items: make(map[string]interface{}),
		mu:    sync.Mutex{},
	}
}

package internal

import (
	"sort"
	"sync"
)

var (
	staticData = []*item{
		{Code: "A12T-4GH7-QPL9-3N4M", Name: "Lettuce", Price: 3.46},
		{Code: "E5T6-9UI3-TH15-QR88", Name: "Peach", Price: 2.99},
		{Code: "TQ4C-VV6T-75ZX-1RMR", Name: "Gala Apple", Price: 3.59},
		{Code: "YRT6-72AS-K736-L4AR", Name: "Green Pepper", Price: 0.79},
	}
)

type item struct {
	Code  string  `json:"code"`
	Name  string  `json:"name"`
	Price float32 `json:"price"`
}

type Store struct {
	mu    sync.RWMutex
	items []*item // sorted by code
}

func NewStore() *Store {
	return &Store{
		items: append([]*item{}, staticData...),
	}
}

func (s *Store) Get(code string) *item {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Binary search for the given item (>=)
	idx := sort.Search(len(s.items), func(i int) bool {
		return s.items[i].Code >= code
	})
	if idx < len(s.items) && s.items[idx].Code == code {
		itCopy := *s.items[idx]
		return &itCopy
	}
	return nil
}

func (s *Store) Put(it item) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Binary search for the given item (>=)
	idx := sort.Search(len(s.items), func(i int) bool {
		return s.items[i].Code >= it.Code
	})

	if idx < len(s.items) && s.items[idx].Code == it.Code {
		// If the item already exists, overwrite it.
		s.items[idx] = &it
	} else {
		// append and sort
		s.items = append(s.items, &it)
		sort.Slice(s.items, func(i, j int) bool {
			return s.items[i].Code < s.items[j].Code
		})
	}
}

func (s *Store) List(cursor string, count int) ([]item, string) {
	if count == 0 {
		// if you ask for nothing, you get nothing...
		return nil, cursor
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	startIdx := 0
	if cursor != "" {
		// Binary search for one beyond the given cursor (>)
		startIdx = sort.Search(len(s.items), func(i int) bool {
			return s.items[i].Code > cursor
		})
	}

	if count > len(s.items)-startIdx {
		count = len(s.items) - startIdx
	}

	if count == 0 {
		// Nothing left after finding the cursor.
		return nil, ""
	}

	ret := make([]item, 0, count)
	for i := 0; i < count; i++ {
		ret = append(ret, *s.items[i+startIdx]) // make a copy of each item
	}

	newCursor := ""
	if startIdx+count < len(s.items) {
		// If there are more items remaining, use the last returned item as the cursor.
		newCursor = ret[count-1].Code
	}

	return ret, newCursor
}

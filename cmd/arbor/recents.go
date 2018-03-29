package main

import (
	"sync"
)

type RecentList struct {
	Recents []string
	index   int
	sync.RWMutex
}

func NewRecents(size int) *RecentList {
	return &RecentList{
		Recents: make([]string, size),
		index:   0,
	}
}

func (r *RecentList) Add(id string) {
	r.Lock()
	defer r.Unlock()
	r.Recents[r.index] = id
	r.index++
	r.index %= len(r.Recents)
}

func (r *RecentList) Data() []string {
	r.RLock()
	defer r.RUnlock()
	res := make([]string, len(r.Recents))
	copy(res, r.Recents)
	return res
}

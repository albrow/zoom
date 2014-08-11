package zoom

import (
	"sync"
)

var mutexMap = make(map[string]*sync.Mutex)
var globalMutexLock sync.Mutex

func getMutexForId(id string) *sync.Mutex {
	globalMutexLock.Lock()
	defer globalMutexLock.Unlock()
	if existing, found := mutexMap[id]; found {
		return existing
	} else {
		// create a new Mutex and add it to the map
		mut := &sync.Mutex{}
		mutexMap[id] = mut
		return mut
	}
}

type Sync struct {
	MutexId string
}

func (s Sync) Lock() {
	if s.MutexId == "" {
		panic("Zoom: attempt to call Lock on a model without an Id")
	}
	getMutexForId(s.MutexId).Lock()
}

func (s Sync) Unlock() {
	if s.MutexId == "" {
		panic("Zoom: attempt to call Unlock on a model without an Id")
	}
	getMutexForId(s.MutexId).Unlock()
}

func (s *Sync) SetMutexId(id string) {
	s.MutexId = id
}

type Syncer interface {
	Lock()
	Unlock()
	SetMutexId(string)
}

package zoom

import (
	"sync"
)

var mutexMap = make(map[string]*sync.Mutex)
var globalMutexLock sync.Mutex

// GetMutexById returns a mutex identified by mutexId. It will always
// return the same mutex given the same mutexId. It can be used to manually
// lock and unlock references to specific redis keys for stricter thread-safety.
func GetMutexById(mutexId string) *sync.Mutex {
	globalMutexLock.Lock()
	defer globalMutexLock.Unlock()
	if existing, found := mutexMap[mutexId]; found {
		return existing
	} else {
		// create a new Mutex and add it to the map
		mut := &sync.Mutex{}
		mutexMap[mutexId] = mut
		return mut
	}
}

// Sync provides a default implementation of Syncer. It can be directly
// embedded into model structs.
type Sync struct {
	MutexId string `json:"-",redis:"-"`
}

func (s Sync) Lock() {
	if s.MutexId == "" {
		panic("Zoom: attempt to call Lock on a model without an Id")
	}
	GetMutexById(s.MutexId).Lock()
}

func (s Sync) Unlock() {
	if s.MutexId == "" {
		panic("Zoom: attempt to call Unlock on a model without an Id")
	}
	GetMutexById(s.MutexId).Unlock()
}

func (s *Sync) SetMutexId(id string) {
	s.MutexId = id
}

// Syncer is an interface meant for controlling accesss to a model in
// a thread-safe manner. Any model that implements Syncer will automatically
// call Lock before being retrieved from the database, so that there can effectively
// only be one active reference to a model (identified by a mutexId) at any given
// time. Zoom will not automatically call Unlock, however, and it is up to the
// user call Unlock when done making changes to a specific model.
type Syncer interface {
	Lock()
	Unlock()
	SetMutexId(string)
}

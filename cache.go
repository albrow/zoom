package zoom

import (
	"github.com/stephenalexbrowne/zoom/cache"
	"github.com/stephenalexbrowne/zoom/redis"
	"reflect"
	"sync"
	"time"
)

var zoomCache *cache.LRUCache

// requested index cache updates will go through a maximum
// of once per model per INDEX_CACHE_THROTTLE duration.
const INDEX_CACHE_THROTTLE time.Duration = 1 * time.Second

var indexCacheLocks map[string]*indexCacheLock = make(map[string]*indexCacheLock)

type cacheValue struct {
	value interface{}
	size  int
}

func newCacheValue(in interface{}) *cacheValue {
	s := int(reflect.TypeOf(in).Size())
	return &cacheValue{in, s}
}

type indexCacheLock struct {
	m     *sync.Mutex
	timer *time.Timer
}

func newIndexCache() *indexCacheLock {
	return &indexCacheLock{
		m:     &sync.Mutex{},
		timer: nil,
	}
}

func (c *cacheValue) Size() int {
	return c.size
}

func ClearCache() {
	zoomCache.Clear()
}

func ScheduleIndexCacheUpdate(modelName string) {
	ic, found := indexCacheLocks[modelName]
	if !found {
		ic = newIndexCache()
		indexCacheLocks[modelName] = ic
	}
	ic.m.Lock()
	if ic.timer == nil {
		ic.timer = time.NewTimer(INDEX_CACHE_THROTTLE)
		go func() {
			updateIndexCache(modelName)
			<-ic.timer.C
			ic.timer = nil
		}()
	} else {
	}
	ic.m.Unlock()
}

func updateIndexCache(modelName string) {

	// get a connection
	conn := pool.Get()
	defer conn.Close()

	// invoke redis driver to get indexed keys and convert to []interface{}
	key := modelName + ":index"
	ids, _ := redis.Strings(conn.Do("smembers", key))

	// iterate through each id. find the corresponding model. append to results.
	results := make([]interface{}, len(ids), len(ids))
	for i, id := range ids {
		m, _ := FindById(modelName, id)
		results[i] = m
	}

	zoomCache.Set(key, newCacheValue(results))
}

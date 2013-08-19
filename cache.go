package zoom

import (
	"fmt"
	"github.com/stephenalexbrowne/zoom/cache"
	"reflect"
)

var modelCache *cache.LRUCache

type cacheValue struct {
	value interface{}
	size  int
}

func newCacheValue(in interface{}) *cacheValue {
	s := int(reflect.TypeOf(in).Size())
	return &cacheValue{in, s}
}

func (c *cacheValue) Size() int {
	return c.size
}

func ClearCache() {
	modelCache.Clear()
}

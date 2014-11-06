package zoom

type idSet struct {
	ids      []string
	pristine bool // true iff no ids have been added yet
}

func newIdSet() *idSet {
	return &idSet{
		pristine: true,
	}
}

func (s *idSet) intersect(ids []string) {
	// in the special case that s is pristine, we don't
	// actually want to do an intersect because intersecting
	// with an empty slice would produce an empty slice. Instead
	// we set the initial state by simply setting s.ids to ids.
	if s.pristine {
		s.ids = ids
		s.pristine = false
		return
	}

	// we'll store the results in a new slice
	results := []string{}

	// memoize all the elements in the new ids so
	// we can avoid looping over every time to see
	// if they contain a particular id
	memo := make(map[string]struct{})
	for _, id := range ids {
		memo[id] = struct{}{}
	}
	for _, id := range s.ids {
		// iterate over each element in old ids (s.ids).
		// this retains order with respect to the old ids.
		if _, found := memo[id]; found {
			results = append(results, id)
		}
	}
	s.ids = results
}

func (s *idSet) union(other *idSet) {
	s.ids = append(s.ids, other.ids...)
}

func (s *idSet) add(ids ...string) {
	s.ids = append(s.ids, ids...)
}

func (s *idSet) reverse() {
	for i, j := 0, len(s.ids)-1; i <= j; i, j = i+1, j-1 {
		s.ids[i], s.ids[j] = s.ids[j], s.ids[i]
	}
}

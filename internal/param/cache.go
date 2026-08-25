package param

import "chemicalprocessdcs/internal/model"

// Cache holds an independent snapshot of the store's params and curves,
// refreshed when the store generation changes. Reads consult only the local
// snapshot, so callers observe a single consistent version even while the
// store is being mutated concurrently.
type Cache struct {
	store  *Store
	gen    uint64
	cached map[string]model.ProcessParams
	curves map[string]model.Curve
}

func NewCache(store *Store) *Cache {
	return &Cache{
		store:  store,
		cached: make(map[string]model.ProcessParams),
		curves: make(map[string]model.Curve),
	}
}

func (c *Cache) Get(id string) (model.ProcessParams, bool) {
	c.refreshIfStale()
	p, ok := c.cached[id]
	return p, ok
}

func (c *Cache) Curve(id string) (model.Curve, bool) {
	c.refreshIfStale()
	curve, ok := c.curves[id]
	return curve, ok
}

func (c *Cache) refreshIfStale() {
	gen := c.store.Generation()
	if gen == c.gen {
		return
	}
	c.refresh(gen)
}

// refresh rebuilds the local snapshot from the store. It reads params and
// curves under the store lock so the two maps are captured together as one
// coherent version.
func (c *Cache) refresh(gen uint64) {
	c.cached = c.store.AllParams()
	c.curves = c.store.AllCurves()
	c.gen = gen
}

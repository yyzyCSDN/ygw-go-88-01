package param

import "chemicalprocessdcs/internal/model"

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
	if c.store.Generation() != c.gen {
		c.refresh()
	}
	p, ok := c.cached[id]
	return p, ok
}

func (c *Cache) refresh() {
	next := make(map[string]model.ProcessParams)
	for id := range c.store.params {
		if p, ok := c.store.Params(id); ok {
			next[id] = p
		}
	}
	nextCurves := make(map[string]model.Curve)
	for id := range c.store.curves {
		if curve, ok := c.store.Curve(id); ok {
			nextCurves[id] = curve
		}
	}
	c.cached = next
	c.curves = nextCurves
	c.gen = c.store.Generation()
}

func (c *Cache) Curve(id string) (model.Curve, bool) {
	if c.store.Generation() != c.gen {
		c.refresh()
	}
	curve, ok := c.curves[id]
	return curve, ok
}

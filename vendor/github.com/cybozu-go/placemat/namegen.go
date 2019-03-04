package placemat

import (
	"fmt"
	"sync"
)

type nameGenerator struct {
	prefix string
	mu     sync.Mutex
	id     int
}

func (g *nameGenerator) New() string {
	g.mu.Lock()
	defer g.mu.Unlock()

	name := fmt.Sprintf("%s%d", g.prefix, g.id)
	g.id++
	return name
}

func (g *nameGenerator) GeneratedNames() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	rv := make([]string, g.id)
	for i := 0; i < g.id; i++ {
		rv[i] = fmt.Sprintf("%s%d", g.prefix, i)
	}
	return rv
}

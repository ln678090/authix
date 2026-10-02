package cache

import (
	"hash/fnv"
	"sync"

	"git.blinkpay.vn/software/authix/internal/persistence"
)

const numShards = 256

type PermissionCache struct {
	shards [numShards]*shard
}

type shard struct {
	mu   sync.RWMutex
	data map[string]*persistence.EffectivePermissions
}

func NewPermissionCache() *PermissionCache {
	c := &PermissionCache{}
	for i := range numShards {
		c.shards[i] = &shard{
			data: make(map[string]*persistence.EffectivePermissions),
		}
	}
	return c
}

func (c *PermissionCache) getShard(key string) *shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return c.shards[h.Sum32()%numShards]
}

func (c *PermissionCache) Get(key string) (*persistence.EffectivePermissions, bool) {
	s := c.getShard(key)
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.data[key]
	return entry, ok
}

func (c *PermissionCache) Set(key string, data *persistence.EffectivePermissions) {
	s := c.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	s.data[key] = data
}

func (c *PermissionCache) Delete(key string) {
	s := c.getShard(key)
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.data, key)
}

func (c *PermissionCache) Clear() {
	for _, s := range c.shards {
		s.mu.Lock()
		s.data = make(map[string]*persistence.EffectivePermissions)
		s.mu.Unlock()
	}
}

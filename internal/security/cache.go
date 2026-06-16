package security

import (
	"sync"
	"time"
)

type subscriptionCacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

type subscriptionCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]subscriptionCacheEntry
}

func newSubscriptionCache(ttl time.Duration) *subscriptionCache {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &subscriptionCache{
		ttl:     ttl,
		entries: make(map[string]subscriptionCacheEntry),
	}
}

func (c *subscriptionCache) get(establishmentID string) (allowed bool, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.entries[establishmentID]
	if !found || time.Now().After(entry.expiresAt) {
		return false, false
	}

	return entry.allowed, true
}

func (c *subscriptionCache) set(establishmentID string, allowed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[establishmentID] = subscriptionCacheEntry{
		allowed:   allowed,
		expiresAt: time.Now().Add(c.ttl),
	}
}

func (c *subscriptionCache) invalidate(establishmentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, establishmentID)
}

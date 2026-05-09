package deepseek

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// PromptCache manages prompt caching for repeated message prefixes.
// DeepSeek supports automatic prompt caching for repeated prefixes.
type PromptCache struct {
	mu     sync.RWMutex
	hashes map[string]string // content hash → cache key
	stats  CacheStats
}

type CacheStats struct {
	Hits      int     `json:"hits"`
	Misses    int     `json:"misses"`
	HitRate   float64 `json:"hitRate"`
	TokensSaved int   `json:"tokensSaved"`
}

func NewPromptCache() *PromptCache {
	return &PromptCache{
		hashes: make(map[string]string),
	}
}

func (c *PromptCache) Key(content string) string {
	c.mu.RLock()
	if key, ok := c.hashes[content]; ok {
		c.mu.RUnlock()
		c.mu.Lock()
		c.stats.Hits++
		c.mu.Unlock()
		return key
	}
	c.mu.RUnlock()

	hash := sha256.Sum256([]byte(content))
	key := hex.EncodeToString(hash[:16])

	c.mu.Lock()
	c.hashes[content] = key
	c.stats.Misses++
	c.mu.Unlock()

	return key
}

func (c *PromptCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	stats := c.stats
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total) * 100
	}
	return stats
}

func (c *PromptCache) Reset() {
	c.mu.Lock()
	c.hashes = make(map[string]string)
	c.stats = CacheStats{}
	c.mu.Unlock()
}

package models

import (
	"container/list"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultShards     = 1024
	MemCheckInterval  = 2 * time.Second  // Reduced interval for faster response to memory pressure
	MaxMemoryPercent  = 0.7 // 70% memory threshold
	EvictionBatchSize = 200  // Increased batch size for more aggressive eviction
)

// ShardedCache implements a sharded cache with CLOCK eviction
type ShardedCache struct {
	shards       []*CacheShard
	shardCount   int
	maxMemoryPct float64
	memUsage     int64
	stopChan     chan struct{}
}

// CacheShard represents a single shard in the cache
type CacheShard struct {
	items     map[string]string
	elements  map[string]*list.Element  // Direct element access for O(1) lookups
	clockHand *list.Element
	itemsList *list.List
	mu        sync.RWMutex
}

// ClockItem represents an item in the CLOCK cache
type ClockItem struct {
	key        string
	referenced bool
	size       int64  // Track size for better memory management
}

// NewCache creates a new sharded cache
func NewCache() *ShardedCache {
	cache := &ShardedCache{
		shards:       make([]*CacheShard, DefaultShards),
		shardCount:   DefaultShards,
		maxMemoryPct: MaxMemoryPercent,
		stopChan:     make(chan struct{}),
	}

	for i := 0; i < DefaultShards; i++ {
		cache.shards[i] = &CacheShard{
			items:     make(map[string]string),
			elements:  make(map[string]*list.Element),  // Initialize elements map
			itemsList: list.New(),
		}
	}

	// Start memory monitor goroutine
	go cache.monitorMemory()

	return cache
}

// Get retrieves a value from the cache
func (c *ShardedCache) Get(key string) (string, bool) {
	shard := c.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	if val, ok := shard.items[key]; ok {
		// O(1) lookup for element using the elements map
		if elem, found := shard.elements[key]; found {
			item := elem.Value.(*ClockItem)
			item.referenced = true
		}
		return val, true
	}
	return "", false
}

// Put adds or updates a value in the cache
func (c *ShardedCache) Put(key, value string) {
	shard := c.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	// Calculate size for memory tracking
	valueSize := int64(len(value))
	keySize := int64(len(key))
	totalSize := keySize + valueSize

	// Check if key already exists
	if oldVal, exists := shard.items[key]; exists {
		// Update value
		shard.items[key] = value
		
		// Update memory usage with size difference
		oldSize := int64(len(oldVal))
		sizeDiff := valueSize - oldSize
		if sizeDiff != 0 {
			atomic.AddInt64(&c.memUsage, sizeDiff)
		}
		
		// Update referenced status using O(1) lookup
		if elem, found := shard.elements[key]; found {
			item := elem.Value.(*ClockItem)
			item.referenced = true
			item.size = totalSize
		}
		return
	}

	// Add new item
	shard.items[key] = value
	
	// Create new clock item with size information
	item := &ClockItem{
		key:        key,
		referenced: true,
		size:       totalSize,
	}
	
	// Add to linked list and store reference in elements map
	element := shard.itemsList.PushBack(item)
	shard.elements[key] = element

	// Initialize clockHand if this is the first item
	if shard.clockHand == nil {
		shard.clockHand = element
	}

	// Update memory usage estimate
	atomic.AddInt64(&c.memUsage, totalSize)
}

// getShard returns the appropriate shard for a key
func (c *ShardedCache) getShard(key string) *CacheShard {
	// Simple hash function
	hash := fnv32(key)
	return c.shards[hash%uint32(c.shardCount)]
}

// fnv32 implements a simple hash function
func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash *= 16777619
		hash ^= uint32(key[i])
	}
	return hash
}

// monitorMemory checks memory usage and triggers eviction if needed
func (c *ShardedCache) monitorMemory() {
	ticker := time.NewTicker(MemCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			// Check if memory usage exceeds threshold
			memRatio := float64(m.Alloc) / float64(m.Sys)
			if memRatio > c.maxMemoryPct {
				// More aggressive eviction when memory pressure is high
				evictionCount := EvictionBatchSize
				if memRatio > 0.85 {
					evictionCount *= 2
				}
				c.evictBatch(evictionCount)
			}
		case <-c.stopChan:
			return
		}
	}
}

// evictBatch evicts a batch of items using the CLOCK algorithm
func (c *ShardedCache) evictBatch(count int) {
	evicted := 0
	// Distribute eviction across shards
	perShardCount := count / c.shardCount
	if perShardCount < 1 {
		perShardCount = 1
	}
	
	for i := 0; i < c.shardCount && evicted < count; i++ {
		shard := c.shards[i]
		evicted += shard.evict(perShardCount)
	}
}

// evict implements the CLOCK algorithm for a single shard
func (s *CacheShard) evict(count int) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.clockHand == nil || s.itemsList.Len() == 0 {
		return 0
	}

	evicted := 0
	attempts := 0
	maxAttempts := s.itemsList.Len() * 2 // Prevent infinite loops
	
	for evicted < count && attempts < maxAttempts {
		attempts++
		item := s.clockHand.Value.(*ClockItem)

		if item.referenced {
			// Give a second chance
			item.referenced = false
			s.clockHand = nextOrFirst(s.clockHand, s.itemsList)
		} else {
			// Evict this item
			next := nextOrFirst(s.clockHand, s.itemsList)
			delete(s.items, item.key)
			delete(s.elements, item.key)  // Clean up elements map
			s.itemsList.Remove(s.clockHand)
			s.clockHand = next
			evicted++

			if s.itemsList.Len() == 0 {
				s.clockHand = nil
				break
			}
		}
	}

	return evicted
}

// nextOrFirst returns the next element or circles back to the first
func nextOrFirst(e *list.Element, l *list.List) *list.Element {
	if e.Next() == nil {
		return l.Front()
	}
	return e.Next()
}

// Close stops all background goroutines
func (c *ShardedCache) Close() {
	close(c.stopChan)
}

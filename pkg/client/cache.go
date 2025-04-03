package client

import (
	"context"
	"sync"
	"time"

	"github.com/example/nfsserver/pkg/api"
)

// AttrCache provides a cache for file attributes
// TODO: Implement a proper attribute cache
type AttrCache struct {
	// Maximum cache size
	maxSize int

	// Time-to-live for cache entries
	ttl time.Duration
}

// AttrCacheEntry represents a cached attribute with expiration time
type AttrCacheEntry struct {
	value      *api.FileAttributes
	expiration time.Time
}

// NewAttrCache creates a new attributes cache
func NewAttrCache(maxSize int, ttl time.Duration) *AttrCache {
	// TODO: Implement actual attribute cache
	return &AttrCache{
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// StorePathAttrs stores attributes for a path
func (c *AttrCache) StorePathAttrs(path string, attrs *api.FileAttributes) {
	// TODO: Implement attribute caching
}

// StoreHandleAttrs stores attributes for a handle
func (c *AttrCache) StoreHandleAttrs(handle []byte, attrs *api.FileAttributes) {
	// TODO: Implement attribute caching
}

// GetPathAttrs retrieves attributes for a path
func (c *AttrCache) GetPathAttrs(path string) (*api.FileAttributes, bool) {
	// TODO: Implement attribute retrieval from cache
	return nil, false
}

// GetHandleAttrs retrieves attributes for a handle
func (c *AttrCache) GetHandleAttrs(handle []byte) (*api.FileAttributes, bool) {
	// TODO: Implement attribute retrieval from cache
	return nil, false
}

type WriteBatchCache struct {
	mu        sync.Mutex
	reqs      []*api.WriteRequest
	totalSize int
	maxSize   int
	client    *Client
}

func NewWriteBatchCache(maxSize int, client *Client) *WriteBatchCache {
	return &WriteBatchCache{
		reqs:    make([]*api.WriteRequest, 0),
		maxSize: maxSize,
		client:  client,
	}
}

func (w *WriteBatchCache) Write(ctx context.Context, fileHandle []byte, offset int64, data []byte, stability int) (int, error) {
	if stability < 0 || stability > 2 {
		stability = 0
	}

	cloned := make([]byte, len(data))
	copy(cloned, data)

	req := &api.WriteRequest{
		FileHandle: fileHandle,
		Credentials: &api.Credentials{
			Uid:    0,
			Gid:    0,
			Groups: []uint32{0},
		},
		Offset:    uint64(offset),
		Data:      cloned,
		Stability: uint32(stability),
	}

	w.mu.Lock()
	w.reqs = append(w.reqs, req)
	w.totalSize += len(data)
	shouldFlush := w.totalSize >= w.maxSize
	w.mu.Unlock()

	if shouldFlush {
		return len(data), w.FlushAll(ctx)
	}

	return len(data), nil
}

// flush when close file or cache is full
// TODO close call flush
func (w *WriteBatchCache) FlushAll(ctx context.Context) error {
	w.mu.Lock()
	batch := w.reqs
	w.reqs = make([]*api.WriteRequest, 0)
	w.totalSize = 0
	w.mu.Unlock()

	if len(batch) == 0 {
		return nil
	}

	callCtx, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()

	_, err := w.client.nfsClient.WriteBatch(callCtx, &api.WriteBatchRequest{
		Writes: batch,
	})

	if err != nil {
		return err
	}

	return nil
}

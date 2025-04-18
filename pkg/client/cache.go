package client

import (
	"context"
	"sync"
	"time"

	"github.com/example/nfsserver/pkg/api"
)

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

	callCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := w.client.nfsClient.WriteBatch(callCtx, &api.WriteBatchRequest{
		Writes: batch,
	})

	if err != nil {
		return err
	}

	return nil
}

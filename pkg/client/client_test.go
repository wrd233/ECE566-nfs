package client

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.ServerAddress != "localhost:2049" {
		t.Errorf("Expected default ServerAddress to be localhost:2049, got %s", config.ServerAddress)
	}
	
	if config.MaxRetries != 3 {
		t.Errorf("Expected default MaxRetries to be 3, got %d", config.MaxRetries)
	}
}

// TestHandleCache is a simplified test for handle cache stubs
func TestHandleCache(t *testing.T) {
	cache := NewHandleCache(100, 1)
	
	// Just test that we can create and call methods on the stub
	if cache.maxSize != 100 {
		t.Errorf("Expected maxSize to be 100, got %d", cache.maxSize)
	}
	
	// Test methods don't panic
	cache.StorePathHandle("/test/path", []byte{1, 2, 3, 4})
	cache.StoreHandlePath([]byte{1, 2, 3, 4}, "/test/path")
	
	// GetHandle should return false since it's stubbed
	_, ok := cache.GetHandle("/test/path")
	if ok {
		t.Error("Stubbed cache.GetHandle unexpectedly returned true")
	}
	
	// GetPath should return false since it's stubbed
	_, ok = cache.GetPath([]byte{1, 2, 3, 4})
	if ok {
		t.Error("Stubbed cache.GetPath unexpectedly returned true")
	}
}
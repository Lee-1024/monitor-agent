package main

import (
	"os"
	"testing"

	pb "monitor-agent/proto"
)

func TestMetricCacheStoreAndFlush(t *testing.T) {
	dir := t.TempDir()
	cache := NewMetricCache(dir, 10)

	req := &pb.MetricsRequest{HostId: "host-a", Timestamp: 123}
	if err := cache.Store(req); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read cache dir failed: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 cache file, got %d", len(files))
	}

	var sent []*pb.MetricsRequest
	flushed, err := cache.Flush(func(req *pb.MetricsRequest) error {
		sent = append(sent, req)
		return nil
	})
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if flushed != 1 {
		t.Fatalf("expected 1 flushed item, got %d", flushed)
	}
	if len(sent) != 1 || sent[0].HostId != "host-a" || sent[0].Timestamp != 123 {
		t.Fatalf("unexpected flushed request: %#v", sent)
	}

	files, err = os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read cache dir after flush failed: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected cache file to be removed, got %d files", len(files))
	}
}

func TestMetricCacheEnforcesMaxFiles(t *testing.T) {
	dir := t.TempDir()
	cache := NewMetricCache(dir, 2)

	for i := int64(0); i < 3; i++ {
		if err := cache.Store(&pb.MetricsRequest{HostId: "host-a", Timestamp: i}); err != nil {
			t.Fatalf("store %d failed: %v", i, err)
		}
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read cache dir failed: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected max 2 cache files, got %d", len(files))
	}
}

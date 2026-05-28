package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	pb "monitor-agent/proto"
)

type MetricCache struct {
	dir      string
	maxFiles int
}

func NewMetricCache(dir string, maxFiles int) *MetricCache {
	if dir == "" {
		dir = "./agent-cache"
	}
	if maxFiles <= 0 {
		maxFiles = 1000
	}
	return &MetricCache{dir: dir, maxFiles: maxFiles}
}

func (c *MetricCache) Store(req *pb.MetricsRequest) error {
	if c == nil || req == nil {
		return nil
	}
	if err := os.MkdirAll(c.dir, 0755); err != nil {
		return err
	}
	if err := c.enforceLimit(); err != nil {
		return err
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%d-%s.json", time.Now().UnixNano(), req.HostId)
	return os.WriteFile(filepath.Join(c.dir, name), data, 0644)
}

func (c *MetricCache) Flush(send func(*pb.MetricsRequest) error) (int, error) {
	if c == nil || send == nil {
		return 0, nil
	}
	files, err := c.files()
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	flushed := 0
	for _, file := range files {
		path := filepath.Join(c.dir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return flushed, err
		}
		var req pb.MetricsRequest
		if err := json.Unmarshal(data, &req); err != nil {
			_ = os.Remove(path)
			continue
		}
		if err := send(&req); err != nil {
			return flushed, err
		}
		if err := os.Remove(path); err != nil {
			return flushed, err
		}
		flushed++
	}
	return flushed, nil
}

func (c *MetricCache) enforceLimit() error {
	files, err := c.files()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for len(files) >= c.maxFiles {
		oldest := filepath.Join(c.dir, files[0].Name())
		if err := os.Remove(oldest); err != nil {
			return err
		}
		files = files[1:]
	}
	return nil
}

func (c *MetricCache) files() ([]os.DirEntry, error) {
	files, err := os.ReadDir(c.dir)
	if err != nil {
		return nil, err
	}
	files = filterJSONFiles(files)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})
	return files, nil
}

func filterJSONFiles(files []os.DirEntry) []os.DirEntry {
	filtered := make([]os.DirEntry, 0, len(files))
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

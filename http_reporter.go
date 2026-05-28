package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	pb "monitor-agent/proto"
)

type HTTPReporter struct {
	baseURL string
	client  *http.Client
}

func NewHTTPReporter(baseURL string, timeout time.Duration) *HTTPReporter {
	return &HTTPReporter{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: timeout},
	}
}

func (r *HTTPReporter) Register(ctx context.Context, req *pb.RegisterRequest) error {
	return r.post(ctx, "/api/v1/agent/register", req)
}

func (r *HTTPReporter) Heartbeat(ctx context.Context, req *pb.HeartbeatRequest) error {
	return r.post(ctx, "/api/v1/agent/heartbeat", req)
}

func (r *HTTPReporter) ReportMetrics(ctx context.Context, req *pb.MetricsRequest) error {
	return r.post(ctx, "/api/v1/agent/metrics", req)
}

func (r *HTTPReporter) post(ctx context.Context, path string, payload interface{}) error {
	if r == nil || r.baseURL == "" {
		return fmt.Errorf("http reporter is not configured")
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http reporter got status %d", resp.StatusCode)
	}
	return nil
}

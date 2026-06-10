package stats

import (
	"sync/atomic"
	"time"
)

type Collector struct {
	startedAt         time.Time
	uploadedBytes     atomic.Int64
	downloadedBytes   atomic.Int64
	activeConnections atomic.Int64
	totalConnections  atomic.Int64
}

type Snapshot struct {
	StartedAt         time.Time `json:"started_at"`
	UploadedBytes     int64     `json:"uploaded_bytes"`
	DownloadedBytes   int64     `json:"downloaded_bytes"`
	ActiveConnections int64     `json:"active_connections"`
	TotalConnections  int64     `json:"total_connections"`
}

func NewCollector() *Collector {
	return &Collector{startedAt: time.Now().UTC()}
}

func (c *Collector) AddUploaded(n int64) {
	if c != nil && n > 0 {
		c.uploadedBytes.Add(n)
	}
}

func (c *Collector) AddDownloaded(n int64) {
	if c != nil && n > 0 {
		c.downloadedBytes.Add(n)
	}
}

func (c *Collector) OpenConnection() {
	if c == nil {
		return
	}
	c.activeConnections.Add(1)
	c.totalConnections.Add(1)
}

func (c *Collector) CloseConnection() {
	if c != nil {
		c.activeConnections.Add(-1)
	}
}

func (c *Collector) Snapshot() Snapshot {
	if c == nil {
		return Snapshot{}
	}
	return Snapshot{
		StartedAt:         c.startedAt,
		UploadedBytes:     c.uploadedBytes.Load(),
		DownloadedBytes:   c.downloadedBytes.Load(),
		ActiveConnections: c.activeConnections.Load(),
		TotalConnections:  c.totalConnections.Load(),
	}
}

// ============================================
// 文件: collector_log.go
// 日志收集器
// ============================================
package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogCollector 日志收集器
type LogCollector struct {
	logPaths []string // 要收集的日志文件路径
	maxLines int      // 每个文件最多收集的行数
}

// NewLogCollector 创建日志收集器
func NewLogCollector(logPaths []string, maxLines int) *LogCollector {
	if maxLines <= 0 {
		maxLines = 100 // 默认每个文件最多100行
	}
	return &LogCollector{
		logPaths: logPaths,
		maxLines: maxLines,
	}
}

// Name 返回采集器名称
func (c *LogCollector) Name() string {
	return "log"
}

// Collect 采集日志
func (c *LogCollector) Collect() (interface{}, error) {
	var allLogs []LogEntry

	for _, logPath := range c.logPaths {
		logs, err := c.collectFromFile(logPath)
		if err != nil {
			continue // 跳过无法读取的文件
		}
		allLogs = append(allLogs, logs...)
	}

	return &LogMetrics{
		Entries: allLogs,
		Count:   len(allLogs),
	}, nil
}

// collectFromFile 从文件收集日志
func (c *LogCollector) collectFromFile(filePath string) ([]LogEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	lineCount := 0

	// 读取文件末尾的N行
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > c.maxLines {
			lines = lines[1:] // 保持只保留最后N行
		}
	}

	// 解析日志行
	for _, line := range lines {
		if lineCount >= c.maxLines {
			break
		}

		entry := c.parseLogLine(filePath, line)
		if entry != nil {
			entries = append(entries, *entry)
			lineCount++
		}
	}

	return entries, scanner.Err()
}

// parseLogLine 解析日志行
func (c *LogCollector) parseLogLine(source, line string) *LogEntry {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	// 简单的日志级别检测
	level := "INFO"
	upperLine := strings.ToUpper(line)
	if strings.Contains(upperLine, "ERROR") || strings.Contains(upperLine, "FATAL") {
		level = "ERROR"
	} else if strings.Contains(upperLine, "WARN") || strings.Contains(upperLine, "WARNING") {
		level = "WARN"
	} else if strings.Contains(upperLine, "DEBUG") {
		level = "DEBUG"
	}

	return &LogEntry{
		Source:    filepath.Base(source),
		Level:     level,
		Message:   line,
		Timestamp: time.Now().Unix(),
		Tags: map[string]string{
			"file": source,
		},
	}
}

// LogEntry 日志条目
type LogEntry struct {
	Source    string            `json:"source"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Timestamp int64             `json:"timestamp"`
	Tags      map[string]string `json:"tags"`
}

// LogMetrics 日志指标
type LogMetrics struct {
	Entries []LogEntry `json:"entries"`
	Count   int        `json:"count"`
}


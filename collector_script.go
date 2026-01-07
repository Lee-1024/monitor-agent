// ============================================
// 文件: collector_script.go
// 自定义脚本执行器
// ============================================
package main

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ScriptExecutor 脚本执行器
type ScriptExecutor struct {
	scripts []ScriptConfig // 要执行的脚本配置
}

// ScriptConfig 脚本配置
type ScriptConfig struct {
	ID          string   `yaml:"id"`          // 脚本ID
	Name        string   `yaml:"name"`        // 脚本名称
	Command     string   `yaml:"command"`      // 命令或脚本路径
	Args        []string `yaml:"args"`        // 参数
	Timeout     int      `yaml:"timeout"`     // 超时时间（秒）
	Interval    int      `yaml:"interval"`    // 执行间隔（秒）
	LastRun     int64    `yaml:"-"`           // 上次执行时间
}

// NewScriptExecutor 创建脚本执行器
func NewScriptExecutor(scripts []ScriptConfig) *ScriptExecutor {
	return &ScriptExecutor{
		scripts: scripts,
	}
}

// Name 返回执行器名称
func (e *ScriptExecutor) Name() string {
	return "script"
}

// Collect 执行脚本并收集结果
func (e *ScriptExecutor) Collect() (interface{}, error) {
	now := time.Now().Unix()
	var results []ScriptResult

	for _, script := range e.scripts {
		// 检查是否需要执行（根据间隔）
		if script.Interval > 0 && script.LastRun > 0 {
			if now-script.LastRun < int64(script.Interval) {
				continue // 还没到执行时间
			}
		}

		result := e.executeScript(script)
		results = append(results, result)

		// 更新最后执行时间
		for i := range e.scripts {
			if e.scripts[i].ID == script.ID {
				e.scripts[i].LastRun = now
				break
			}
		}
	}

	return &ScriptMetrics{
		Results: results,
		Count:   len(results),
	}, nil
}

// executeScript 执行单个脚本
func (e *ScriptExecutor) executeScript(config ScriptConfig) ScriptResult {
	startTime := time.Now()
	timeout := time.Duration(config.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second // 默认30秒超时
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// 根据操作系统选择shell
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, "cmd", "/c", config.Command)
	} else {
		// Unix系统，尝试使用sh执行
		cmd = exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("%s %s", config.Command, joinArgs(config.Args)))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	result := ScriptResult{
		ScriptID:  config.ID,
		ScriptName: config.Name,
		Timestamp:  startTime.Unix(),
		Duration:   duration.Milliseconds(),
		Output:     stdout.String(),
		Error:      stderr.String(),
	}

	if err != nil {
		result.Success = false
		result.ExitCode = -1
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		}
		result.Error = err.Error()
		if stderr.String() != "" {
			result.Error = stderr.String() + "\n" + result.Error
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	return result
}

// joinArgs 连接参数
func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return fmt.Sprintf("'%s'", strings.Join(args, "' '"))
}

// ScriptResult 脚本执行结果
type ScriptResult struct {
	ScriptID  string `json:"script_id"`
	ScriptName string `json:"script_name"`
	Timestamp int64  `json:"timestamp"`
	Success   bool   `json:"success"`
	Output    string `json:"output"`
	Error     string `json:"error"`
	ExitCode  int    `json:"exit_code"`
	Duration  int64  `json:"duration_ms"`
}

// ScriptMetrics 脚本指标
type ScriptMetrics struct {
	Results []ScriptResult `json:"results"`
	Count   int           `json:"count"`
}


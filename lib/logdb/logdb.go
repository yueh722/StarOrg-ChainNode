package logdb

import (
	"errors"
	"io"
)

// Record 是 LogDB 层存储的最小单位：key + nodeid + value(bytes)
// value 由上层（Repo）决定序列化格式，比如 JSON。
type Record struct {
	Key    string
	NodeID string
	Value  []byte
}

// LogDB：存储引擎最小接口
// - Append：追加一条记录（事件溯源）
// - Replay：从磁盘/存储回放所有记录（用于重建索引）
// - LatestByKey：按 key 取最新记录（不要求 O(1)，但引擎可优化）
// - ScanByNode：按 node 回放/扫描该节点的记录
type LogDB interface {
	Append(key, nodeID string, value []byte) error
	Replay(days []string, fn func(rec Record) error) error
	LatestByKey(key string) (*Record, error)
	ScanByNode(nodeID string, days []string, fn func(rec Record) error) error
	io.Closer
}

var (
	ErrCommandNotFound = errors.New("command not found")
	ErrInvalidResult   = errors.New("invalid command result")
)

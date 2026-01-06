package cmdlogstore

import (
	"io"
	"time"
)

type Record struct {
	Day    string // YYYY-MM-DD（来源于文件名）
	Time   int64  // UnixMilli（来自每行内容）
	Key    string
	NodeID string
	Value  []byte
}

// RangeQuery：更贴“从后往前扫一段时间”
// - NDays==0 => 扫描所有日志文件（列目录）
// - NDays>0  => 从 StartDay 往前扫 NDays 天（按天倒序），遇到文件不存在就终止
type RangeQuery struct {
	StartDay string // YYYY-MM-DD；空则默认今天（loc 决定）
	NDays    int    // 0 表示 all

	NodeID string
	Key    string

	Limit int // <=0 不限制（注意：本实现是“边扫边回调”，并不保证严格最近 N 条）
	Loc   *time.Location
}

type Option func(*RangeQuery)

func WithStartDay(day string) Option {
	return func(q *RangeQuery) { q.StartDay = day }
}

// WithNDays: 从 StartDay 往前扫 NDays 天；n==0 表示扫全部
func WithNDays(n int) Option {
	return func(q *RangeQuery) { q.NDays = n }
}

func WithRecentDays(n int, loc *time.Location) Option {
	return func(q *RangeQuery) {
		q.NDays = n
		q.Loc = loc
		q.StartDay = "" // 让引擎用“今天”作为 StartDay
	}
}

func WithNode(nodeID string) Option {
	return func(q *RangeQuery) { q.NodeID = nodeID }
}

func WithKey(key string) Option {
	return func(q *RangeQuery) { q.Key = key }
}

func WithLimit(n int) Option {
	return func(q *RangeQuery) { q.Limit = n }
}

// LogDB：存储引擎最小接口
type LogDBRange interface {
	Append(key, nodeID string, value []byte) error
	Latest(opts ...Option) (Record, error)
	Scan(fn func(rec Record) error, opts ...Option) error
	io.Closer
}

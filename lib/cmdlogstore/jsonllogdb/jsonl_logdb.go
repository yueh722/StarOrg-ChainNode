package jsonllogdb

import (
	"bufio"
	logdb "chain/lib/cmdlogstore"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrInvalidDay   = errors.New("invalid day format, want YYYY-MM-DD")
	ErrNotFound     = errors.New("not found")
	errStop         = errors.New("stop")
)

// 每行 JSONL：新增 time 字段
type jsonlLine struct {
	Time     int64  `json:"time"` // UnixMilli
	Key      string `json:"key"`
	NodeID   string `json:"node_id"`
	ValueB64 string `json:"value_b64"`
}

type DailyJSONL struct {
	mu  sync.Mutex
	dir string
	rl  *rotatelogs.RotateLogs
}

func NewDailyJSONL(dir string, retentionDays int) (*DailyJSONL, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	pattern := filepath.Join(dir, "%Y-%m-%d.jsonl")
	opts := []rotatelogs.Option{
		rotatelogs.WithRotationTime(24 * time.Hour),
		rotatelogs.WithLinkName(filepath.Join(dir, "current.jsonl")),
		rotatelogs.WithClock(rotatelogs.Local),
	}
	if retentionDays > 0 {
		opts = append(opts, rotatelogs.WithMaxAge(time.Duration(retentionDays)*24*time.Hour))
	}

	rl, err := rotatelogs.New(pattern, opts...)
	if err != nil {
		return nil, err
	}

	return &DailyJSONL{dir: dir, rl: rl}, nil
}

func (d *DailyJSONL) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.rl != nil {
		return d.rl.Close()
	}
	return nil
}

// Append：写入永远落到“当前时间所属日文件”（rotatelogs 控制）
// 行内写入 time，用于读取后排序
func (d *DailyJSONL) Append(key, nodeID string, value []byte) error {
	if key == "" || nodeID == "" {
		return ErrInvalidInput
	}

	line := jsonlLine{
		Time:     time.Now().UnixMilli(),
		Key:      key,
		NodeID:   nodeID,
		ValueB64: base64.StdEncoding.EncodeToString(value),
	}
	b, err := json.Marshal(line)
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, err := d.rl.Write(b); err != nil {
		return err
	}
	if _, err := d.rl.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

// Scan：按 RangeQuery “从后往前”扫描天文件
// - NDays==0：列目录所有天文件（升序列出），再倒序扫描（最新 -> 最旧）
// - NDays>0：从 StartDay 往前推 NDays 天，按天倒序；遇到文件不存在 => 终止（符合你的要求）
func (d *DailyJSONL) Scan(fn func(rec logdb.Record) error, opts ...logdb.Option) error {
	q := &logdb.RangeQuery{}
	for _, opt := range opts {
		if opt != nil {
			opt(q)
		}
	}

	loc := q.Loc
	if loc == nil {
		loc = time.Local
	}

	days, stopOnMissing, err := d.buildDays(q, loc)
	if err != nil {
		return err
	}

	limit := q.Limit
	seen := 0

	// 注意：days 已经是 “从最新到最旧” 的顺序
	for _, day := range days {
		p := d.dayPath(day)
		err := replayFile(p, day, func(rec logdb.Record) error {
			// 过滤：node / key
			if q.NodeID != "" && rec.NodeID != q.NodeID {
				return nil
			}
			if q.Key != "" && rec.Key != q.Key {
				return nil
			}

			if err := fn(rec); err != nil {
				return err
			}
			if limit > 0 {
				seen++
				if seen >= limit {
					return errStop
				}
			}
			return nil
		})

		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if stopOnMissing {
					// 动态回溯模式：一旦缺文件就终止
					return nil
				}
				// 全量模式：目录里列出来的文件理论上不该 NotExist，保险起见 continue
				continue
			}
			if errors.Is(err, errStop) {
				return nil
			}
			return err
		}
	}

	return nil
}

// Latest：在扫描范围内取“最后读到的那条匹配记录”
// （因为我们天是倒序扫描，但文件内是正序读，所以它不保证严格最新；
//
//	你现在有 Time 字段，想严格最新可以在上层收集后按 Time 排序取最大。）
func (d *DailyJSONL) Latest(opts ...logdb.Option) (logdb.Record, error) {
	q := &logdb.RangeQuery{}
	for _, opt := range opts {
		if opt != nil {
			opt(q)
		}
	}
	if q.Key == "" {
		return logdb.Record{}, fmt.Errorf("%w: Latest requires WithKey(key)", ErrInvalidInput)
	}

	var last *logdb.Record
	err := d.Scan(func(rec logdb.Record) error {
		tmp := rec
		last = &tmp
		return nil
	}, opts...)
	if err != nil {
		return logdb.Record{}, err
	}
	if last == nil {
		return logdb.Record{}, ErrNotFound
	}
	return *last, nil
}

// ---------- internal ----------

func (d *DailyJSONL) dayPath(day string) string {
	return filepath.Join(d.dir, day+".jsonl")
}

// buildDays：根据 RangeQuery 生成待扫描的 days（顺序：最新->最旧）
// stopOnMissing:
// - true：NDays>0 动态回溯模式，遇到缺文件要终止
// - false：NDays==0 全量模式
func (d *DailyJSONL) buildDays(q *logdb.RangeQuery, loc *time.Location) (days []string, stopOnMissing bool, err error) {
	if q.NDays == 0 {
		// 全量：列目录所有文件，再倒序（最新->最旧）
		all, err := d.listAllDays()
		if err != nil {
			return nil, false, err
		}
		// all 是升序，倒序扫描
		for i := len(all) - 1; i >= 0; i-- {
			days = append(days, all[i])
		}
		return days, false, nil
	}

	// 动态回溯：从 start 往前 NDays 天
	start := q.StartDay
	if start == "" {
		start = time.Now().In(loc).Format("2006-01-02")
	}
	if !isDay(start) {
		return nil, false, ErrInvalidDay
	}

	// 生成 NDays 个 day（最新->最旧）
	t0, err := time.ParseInLocation("2006-01-02", start, loc)
	if err != nil {
		return nil, false, ErrInvalidDay
	}
	for i := 0; i < q.NDays; i++ {
		day := t0.AddDate(0, 0, -i).Format("2006-01-02")
		days = append(days, day)
	}
	return days, true, nil
}

func (d *DailyJSONL) listAllDays() ([]string, error) {
	ents, err := os.ReadDir(d.dir)
	if err != nil {
		return nil, err
	}
	var days []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		day := strings.TrimSuffix(name, ".jsonl")
		if isDay(day) {
			days = append(days, day)
		}
	}
	sort.Strings(days) // 升序
	return days, nil
}

func isDay(s string) bool {
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return false
	}
	for i := 0; i < 10; i++ {
		if i == 4 || i == 7 {
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func replayFile(path string, day string, fn func(rec logdb.Record) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)

	for sc.Scan() {
		raw := sc.Bytes()
		if len(raw) == 0 {
			continue
		}
		var l jsonlLine
		if err := json.Unmarshal(raw, &l); err != nil {
			continue
		}
		val, err := base64.StdEncoding.DecodeString(l.ValueB64)
		if err != nil {
			continue
		}
		if l.Key == "" || l.NodeID == "" {
			continue
		}
		rec := logdb.Record{
			Day:    day,
			Time:   l.Time,
			Key:    l.Key,
			NodeID: l.NodeID,
			Value:  val,
		}
		if err := fn(rec); err != nil {
			return err
		}
	}
	return sc.Err()
}

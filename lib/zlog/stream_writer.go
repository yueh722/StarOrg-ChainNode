package zlog

import (
	"chain/lib/queuestore"
	"chain/utils"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type StreamWriter struct {
	enc      zapcore.Encoder
	minLevel zapcore.LevelEnabler

	store  queuestore.QueueStore
	nodeID string
	seq    uint64

	mu sync.Mutex
}

func NewStreamWriter(nodeID string, store queuestore.QueueStore) (*StreamWriter, error) {
	if store == nil {
		return nil, errors.New("store nil")
	}

	minLevel := zapcore.ErrorLevel
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = ""
	encCfg.EncodeTime = nil

	//encCfg.TimeKey = "time"
	//encCfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")

	return &StreamWriter{
		enc:      zapcore.NewJSONEncoder(encCfg), // use json format for saving log into database
		minLevel: minLevel,
		store:    store,
		nodeID:   nodeID,
	}, nil
}

func (c *StreamWriter) Enabled(lvl zapcore.Level) bool { return c.minLevel.Enabled(lvl) }

func (c *StreamWriter) With(fields []zapcore.Field) zapcore.Core {
	c.mu.Lock()
	enc2 := c.enc.Clone()
	c.mu.Unlock()

	for i := range fields {
		fields[i].AddTo(enc2)
	}
	return &StreamWriter{
		enc:      enc2,
		minLevel: c.minLevel,
		store:    c.store,
		nodeID:   c.nodeID,
		seq:      atomic.LoadUint64(&c.seq),
	}
}

func (c *StreamWriter) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

var levelPriority = map[string]int{
	"DEBUG":   0,
	"INFO":    1,
	"INF":     1,
	"WARN":    2,
	"WARNING": 2,
	"ERROR":   3,
	"ERR":     3,
	"FATAL":   4,
	"PANIC":   5,
}

func extractLevel(buf string) (string, bool) {
	upper := strings.ToUpper(buf)

	for level := range levelPriority {
		if strings.Contains(upper, " "+level+" ") {
			return level, true
		}
	}
	return "", false
}

func shouldKeepLog(buf string) bool {
	level, ok := extractLevel(buf)
	if !ok {
		return false
	}

	return levelPriority[level] >= levelPriority["ERROR"]
}

func (c *StreamWriter) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if !c.Enabled(ent.Level) {
		return nil
	}

	// encode zap json
	c.mu.Lock()
	buf, err := c.enc.EncodeEntry(ent, fields)
	c.mu.Unlock()
	if err != nil {
		return err
	}
	defer buf.Free()

	if !shouldKeepLog(buf.String()) {
		return nil
	}

	raw := strings.TrimRight(string(buf.Bytes()), "\n")

	seq := atomic.AddUint64(&c.seq, 1)
	withSeq, err := injectSeq(raw, seq)
	if err != nil {
		withSeq = raw
	}

	t := ent.Time
	if t.IsZero() {
		t = time.Now().Local()
	}

	e := queuestore.PersistedEvent{
		EventID: utils.UUID(),
		NodeID:  c.nodeID,
		Seq:     seq,
		Level:   ent.Level.String(),
		Time:    t.UnixMilli(),
		JSON:    withSeq,
	}

	// write zlog into store
	return c.store.Put(e)
}

func (c *StreamWriter) Sync() error { return nil }

func injectSeq(raw string, seq uint64) (string, error) {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", err
	}
	m["seq"] = seq
	out, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

package log

import (
	"chain/lib/queuestore"
	"encoding/json"
	"errors"
	"strconv"
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
	encCfg.TimeKey = "time"
	encCfg.EncodeTime = zapcore.EpochMillisTimeEncoder

	return &StreamWriter{
		enc:      zapcore.NewJSONEncoder(encCfg),
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

	raw := strings.TrimRight(string(buf.Bytes()), "\n")

	seq := atomic.AddUint64(&c.seq, 1)
	withSeq, err := injectSeq(raw, seq)
	if err != nil {
		withSeq = raw
	}

	t := ent.Time
	if t.IsZero() {
		t = time.Now()
	}

	e := queuestore.PersistedEvent{
		NodeID:  c.nodeID,
		Seq:     seq,
		EventID: c.nodeID + ":" + strconv.FormatUint(seq, 10),
		Level:   ent.Level.String(),
		Time:    t.UnixMilli(),
		JSON:    withSeq,
	}

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

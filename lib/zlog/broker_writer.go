package zlog

import (
	"chain/lib/queuestore"
	"strconv"
	"sync/atomic"

	"go.uber.org/zap/zapcore"
)

type brokerCore struct {
	broker *queuestore.Broker
	name   string
	enc    zapcore.Encoder
	level  zapcore.Level
	seq    uint64
}

func (bc *brokerCore) Enabled(level zapcore.Level) bool {
	return level >= bc.level
}

func (bc *brokerCore) With(fields []zapcore.Field) zapcore.Core {
	clone := bc.clone()
	for _, field := range fields {
		field.AddTo(clone.enc)
	}
	return clone
}

func (bc *brokerCore) Check(entry zapcore.Entry, checkedEntry *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if bc.Enabled(entry.Level) {
		return checkedEntry.AddCore(entry, bc)
	}
	return checkedEntry
}

func (bc *brokerCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	if bc.broker == nil {
		return nil
	}

	buf, err := bc.encodeEntry(entry, fields)
	if err != nil {
		return err
	}

	seq := atomic.AddUint64(&bc.seq, 1)

	bc.broker.Publish(queuestore.PersistedEvent{
		NodeID:  bc.name,
		Seq:     seq,
		EventID: bc.name + ":" + strconv.FormatUint(seq, 10),
		Level:   entry.Level.String(),
		Time:    entry.Time.UnixMilli(),
		JSON:    string(buf),
	})

	return nil
}

func (bc *brokerCore) Sync() error {
	return nil
}

func (bc *brokerCore) clone() *brokerCore {
	return &brokerCore{
		broker: bc.broker,
		name:   bc.name,
		enc:    bc.enc.Clone(),
		level:  bc.level,
	}
}

func (bc *brokerCore) encodeEntry(entry zapcore.Entry, fields []zapcore.Field) ([]byte, error) {

	buf, err := bc.enc.EncodeEntry(entry, fields)
	if err != nil {
		return nil, err
	}

	result := buf.Bytes()

	if pool, ok := bc.enc.(zapcore.ObjectEncoder); ok {
		if put, ok := pool.(interface{ Free() }); ok {
			put.Free()
		}
	}

	return result, nil
}

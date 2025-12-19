package jsonllogdb

import (
	"Chain/lib/logdb"
	"bufio"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type jsonlLine struct {
	Key      string `json:"key"`
	NodeID   string `json:"node_id"`
	ValueB64 string `json:"value_b64"` //base64 format
}

type JSONLLogDB struct {
	mu   sync.Mutex
	path string
	f    *os.File
	w    *bufio.Writer
}

func NewJSONLLogDB(path string) (*JSONLLogDB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &JSONLLogDB{path: path, f: f, w: bufio.NewWriterSize(f, 64*1024)}, nil
}

func (d *JSONLLogDB) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.w != nil {
		_ = d.w.Flush()
	}
	if d.f != nil {
		return d.f.Close()
	}
	return nil
}

func (d *JSONLLogDB) Append(key, nodeID string, value []byte) error {
	if key == "" || nodeID == "" {
		return logdb.ErrInvalidResult
	}
	line := jsonlLine{
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

	if _, err := d.w.Write(b); err != nil {
		return err
	}
	if err := d.w.WriteByte('\n'); err != nil {
		return err
	}
	if err := d.w.Flush(); err != nil {
		return err
	}
	return d.f.Sync()
}

func (d *JSONLLogDB) Replay(fn func(rec logdb.Record) error) error {
	rf, err := os.Open(d.path)
	if err != nil {
		return err
	}
	defer rf.Close()

	sc := bufio.NewScanner(rf)
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
		if err := fn(logdb.Record{Key: l.Key, NodeID: l.NodeID, Value: val}); err != nil {
			return err
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	return nil
}

func (d *JSONLLogDB) LatestByKey(key string) (*logdb.Record, error) {
	if key == "" {
		return nil, logdb.ErrInvalidResult
	}
	var found *logdb.Record
	err := d.Replay(func(rec logdb.Record) error {
		if rec.Key == key {
			tmp := rec
			found = &tmp
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, logdb.ErrCommandNotFound
	}
	return found, nil
}

func (d *JSONLLogDB) ScanByNode(nodeID string, fn func(rec logdb.Record) error) error {
	if nodeID == "" {
		return logdb.ErrInvalidResult
	}
	return d.Replay(func(rec logdb.Record) error {
		if rec.NodeID == nodeID {
			return fn(rec)
		}
		return nil
	})
}

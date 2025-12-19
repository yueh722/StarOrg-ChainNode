package commandlog

import (
	"Chain/lib/logdb"
	"Chain/rpc/common"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type CommandStatus string

const (
	CmdAccepted CommandStatus = "accepted" // received, will execute
	CmdRunning  CommandStatus = "running"  // executing (optional progress updates)
	CmdSuccess  CommandStatus = "success"
	CmdFailed   CommandStatus = "failed"
)

type CommandResult struct {
	CommandID string         `json:"command_id"`
	Cmd       common.CmdType `json:"cmd"`
	NodeID    string         `json:"node_id"`
	Status    CommandStatus  `json:"status"`

	Message string `json:"message,omitempty"` // human readable
	Error   string `json:"error,omitempty"`   // non-empty when failed

	ReportedAt int64 `json:"reported_at"`           // report time (always)
	StartedAt  int64 `json:"started_at,omitempty"`  // optional
	FinishedAt int64 `json:"finished_at,omitempty"` // optional

	// optional: structured output, logs, extra metadata, etc.
	Output json.RawMessage `json:"output,omitempty"`
}

type CommandEventRepo interface {
	Update(r CommandResult) error
	Complete(commandID string, success bool, message string, errStr string, output json.RawMessage) (CommandResult, error)

	GetLatest(commandID string) (CommandResult, error)
	ListLatestByNode(nodeID string) ([]CommandResult, error)

	History(commandID string) ([]CommandResult, error)
	HistoryFromDisk(commandID string) ([]CommandResult, error)

	Close() error
}

type CommandRepo struct {
	db           logdb.LogDB
	mu           sync.RWMutex
	latestByID   map[string]CommandResult
	latestByNode map[string]map[string]CommandResult
	historyByID  map[string][]CommandResult
}

func NewCommandRepo(db logdb.LogDB) (*CommandRepo, error) {
	r := &CommandRepo{
		db:           db,
		latestByID:   make(map[string]CommandResult),
		latestByNode: make(map[string]map[string]CommandResult),
		historyByID:  make(map[string][]CommandResult),
	}

	// build memory record of history
	if err := db.Replay(func(rec logdb.Record) error {
		var e CommandResult
		if err := json.Unmarshal(rec.Value, &e); err != nil {
			return nil // 日志容错：坏行跳过
		}
		if e.CommandID == "" {
			return nil
		}
		r.apply(e)
		return nil
	}); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *CommandRepo) Close() error { return r.db.Close() }

func (r *CommandRepo) Update(e CommandResult) error {
	if err := validateBasic(e); err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	if e.ReportedAt == 0 {
		e.ReportedAt = now
	}
	if e.Status == CmdRunning && e.StartedAt == 0 {
		e.StartedAt = now
	}

	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if err := r.db.Append(e.CommandID, e.NodeID, b); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.applyLocked(e)
	return nil
}

func (r *CommandRepo) Complete(commandID string, success bool, message string, errStr string, output json.RawMessage) (CommandResult, error) {
	if commandID == "" {
		return CommandResult{}, fmt.Errorf("%w: command_id empty", logdb.ErrInvalidResult)
	}

	// 先读最新状态（用内存索引更快；若索引没有也可 db.LatestByKey）
	r.mu.RLock()
	cur, ok := r.latestByID[commandID]
	r.mu.RUnlock()

	if !ok {
		rec, err := r.db.LatestByKey(commandID)
		if err != nil {
			return CommandResult{}, err
		}
		if err := json.Unmarshal(rec.Value, &cur); err != nil || cur.CommandID == "" {
			return CommandResult{}, logdb.ErrCommandNotFound
		}
	}

	now := time.Now().UnixMilli()
	cur.ReportedAt = now
	if cur.StartedAt == 0 {
		cur.StartedAt = now
	}
	cur.FinishedAt = now
	cur.Message = message
	cur.Output = output

	if success {
		cur.Status = CmdSuccess
		cur.Error = ""
	} else {
		cur.Status = CmdFailed
		cur.Error = errStr
	}

	b, err := json.Marshal(cur)
	if err != nil {
		return CommandResult{}, err
	}
	if err := r.db.Append(cur.CommandID, cur.NodeID, b); err != nil {
		return CommandResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.applyLocked(cur)
	return cur, nil
}

func (r *CommandRepo) GetLatest(commandID string) (CommandResult, error) {
	r.mu.RLock()
	v, ok := r.latestByID[commandID]
	r.mu.RUnlock()
	if ok {
		return v, nil
	}

	rec, err := r.db.LatestByKey(commandID)
	if err != nil {
		return CommandResult{}, err
	}
	var e CommandResult
	if err := json.Unmarshal(rec.Value, &e); err != nil || e.CommandID == "" {
		return CommandResult{}, logdb.ErrCommandNotFound
	}
	return e, nil
}

func (r *CommandRepo) ListLatestByNode(nodeID string) ([]CommandResult, error) {
	r.mu.RLock()
	m := r.latestByNode[nodeID]
	r.mu.RUnlock()

	if m != nil {
		out := make([]CommandResult, 0, len(m))
		for _, v := range m {
			out = append(out, v)
		}
		return out, nil
	}

	last := map[string]CommandResult{}
	if err := r.db.ScanByNode(nodeID, func(rec logdb.Record) error {
		var e CommandResult
		if err := json.Unmarshal(rec.Value, &e); err != nil {
			return nil
		}
		if e.CommandID != "" {
			last[e.CommandID] = e
		}
		return nil
	}); err != nil {
		return nil, err
	}

	out := make([]CommandResult, 0, len(last))
	for _, v := range last {
		out = append(out, v)
	}
	return out, nil
}

func (r *CommandRepo) History(commandID string) ([]CommandResult, error) {
	r.mu.RLock()
	h, ok := r.historyByID[commandID]
	r.mu.RUnlock()

	if ok {
		out := make([]CommandResult, len(h))
		copy(out, h)
		return out, nil
	}
	return r.HistoryFromDisk(commandID)
}

func (r *CommandRepo) HistoryFromDisk(commandID string) ([]CommandResult, error) {
	if commandID == "" {
		return nil, fmt.Errorf("%w: command_id empty", logdb.ErrInvalidResult)
	}

	var out []CommandResult
	err := r.db.Replay(func(rec logdb.Record) error {
		if rec.Key != commandID {
			return nil
		}
		var e CommandResult
		if err := json.Unmarshal(rec.Value, &e); err != nil {
			return nil
		}
		out = append(out, e)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, logdb.ErrCommandNotFound
	}
	return out, nil
}

// --- internal helpers ---

func validateBasic(e CommandResult) error {
	if e.CommandID == "" || e.NodeID == "" || e.Cmd == "" || e.Status == "" {
		return fmt.Errorf("%w: command_id/node_id/cmd/status required", logdb.ErrInvalidResult)
	}
	return nil
}

func (r *CommandRepo) apply(e CommandResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.applyLocked(e)
}

func (r *CommandRepo) applyLocked(e CommandResult) {
	r.latestByID[e.CommandID] = e
	if _, ok := r.latestByNode[e.NodeID]; !ok {
		r.latestByNode[e.NodeID] = make(map[string]CommandResult)
	}
	r.latestByNode[e.NodeID][e.CommandID] = e
	r.historyByID[e.CommandID] = append(r.historyByID[e.CommandID], e)
}

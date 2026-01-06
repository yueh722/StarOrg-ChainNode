package commandlog

import (
	"chain/lib/cmdlogstore"
	"chain/rpc/common"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrInvalidCommand  = errors.New("invalid command")
	ErrCommandIdEmpty  = errors.New("command id empty")
	ErrCommandNotFound = errors.New("command not found")

	DefaultLoadLogDays = 30
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

type CompletedRoutine func(err error)

type CommandEventRepo interface {
	Update(r CommandResult, fn CompletedRoutine) error
	Complete(commandID string, success bool, message string, errStr string, output json.RawMessage) (CommandResult, error)

	GetCommand(commandID string) (CommandResult, error)
	ListCommandByNode(nodeID string) ([]CommandResult, error)

	//HistoryFromDiskByNode(nodeID string) ([]CommandResult, error)
	Reset() error
	Close() error
}

type CommandRepo struct {
	store         cmdlogstore.CommandLogStore
	mu            sync.RWMutex
	latestByID    map[string]CommandResult
	latestByNode  map[string]map[string]CommandResult // node_name -> map: command_id -> commandResult
	historyByID   map[string][]CommandResult
	activatedCmds map[string]CompletedRoutine
}

func NewCommandRepo(store cmdlogstore.CommandLogStore) (*CommandRepo, error) {
	r := &CommandRepo{
		store:         store,
		latestByID:    make(map[string]CommandResult),
		latestByNode:  make(map[string]map[string]CommandResult),
		historyByID:   make(map[string][]CommandResult),
		activatedCmds: make(map[string]CompletedRoutine),
	}

	// load all history records of command log
	if err := store.Scan(func(rec cmdlogstore.CommandLogRow) error {
		var e CommandResult
		if err := json.Unmarshal(rec.Value, &e); err != nil {
			return nil // 日志容错：坏行跳过
		}
		if e.CommandID == "" {
			return nil
		}
		r.apply(e)
		return nil
	}, cmdlogstore.Query{}); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *CommandRepo) Close() error { return r.store.Close() }

func (r *CommandRepo) Update(e CommandResult, completedRoutine CompletedRoutine) error {

	var err error
	defer func() {
		if err != nil {
			completedRoutine(err)
		}
	}()

	if err = validateBasic(e); err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	if e.ReportedAt == 0 {
		e.ReportedAt = now
	}
	if e.Status == CmdRunning && e.StartedAt == 0 {
		e.StartedAt = now
	}

	var b []byte
	b, err = json.Marshal(e)
	if err != nil {
		return err
	}
	if err = r.store.Append(e.CommandID, e.NodeID, b); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.applyLocked(e)
	r.activatedCmds[e.CommandID] = completedRoutine

	return nil
}

func (r *CommandRepo) Complete(commandID string, success bool, message string, errStr string, output json.RawMessage) (CommandResult, error) {
	cur, err := r.GetCommand(commandID)
	if err != nil {
		// command does not exist in the memory, ignore status update.
		return cur, err
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
	if err := r.store.Append(cur.CommandID, cur.NodeID, b); err != nil {
		return CommandResult{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.applyLocked(cur)

	completeFn, ok := r.activatedCmds[commandID]
	if ok && completeFn != nil {

		if success {
			completeFn(nil)
		} else {
			completeFn(errors.New(errStr))
		}

		delete(r.activatedCmds, commandID)
	}

	return cur, nil
}

func (r *CommandRepo) GetCommand(commandID string) (CommandResult, error) {
	if commandID == "" {
		return CommandResult{}, ErrCommandIdEmpty
	}

	r.mu.RLock()
	v, ok := r.latestByID[commandID]
	r.mu.RUnlock()
	if ok {
		return v, nil
	}

	return CommandResult{}, ErrInvalidCommand
}

func (r *CommandRepo) ListCommandByNode(nodeID string) ([]CommandResult, error) {
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
	if err := r.store.Scan(func(rec cmdlogstore.CommandLogRow) error {
		var e CommandResult
		if err := json.Unmarshal(rec.Value, &e); err != nil {
			return nil
		}
		if e.CommandID != "" {
			last[e.CommandID] = e
		}
		return nil
	}, cmdlogstore.Query{NodeID: nodeID}); err != nil {
		return nil, err
	}

	out := make([]CommandResult, 0, len(last))
	for _, v := range last {
		out = append(out, v)
	}
	return out, nil
}

func (r *CommandRepo) Reset() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.latestByNode = make(map[string]map[string]CommandResult)
	r.historyByID = make(map[string][]CommandResult)
	r.activatedCmds = make(map[string]CompletedRoutine)
	return r.store.Reset()
}

// --- internal helpers ---

func validateBasic(e CommandResult) error {
	if e.CommandID == "" || e.NodeID == "" || e.Cmd == "" || e.Status == "" {
		return fmt.Errorf("%w: command_id/node_id/cmd/status required", ErrInvalidCommand)
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

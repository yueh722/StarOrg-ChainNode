package command_manager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrCommandNotFound   = errors.New("command not found")
	ErrCommandDuplicated = errors.New("command already exists")
	ErrCommandTimeout    = errors.New("command timeout")
)

// Result 是最终从 SyncMultipleCommands 读出的结果
type Result struct {
	ID  string
	Err error
}

type ActivateOptions struct {
	ID      string
	Timeout time.Duration
	StartFn func(ctx context.Context, id string) error
}

// CmdManager 管理命令生命周期
type CmdManager struct {
	mu   sync.Mutex
	cmds map[string]*cmdState
}

type cmdState struct {
	id string

	// cmdErr channel 用于传递命令的错误
	cmdErr chan error

	finalErr chan error

	once sync.Once
}

func NewCmdManager() *CmdManager {
	return &CmdManager{
		cmds: make(map[string]*cmdState),
	}
}

// ActivateCommand：注册命令 -> 发起 StartFn -> 等待 RPCCompleteCommand 或超时
func (m *CmdManager) ActivateCommand(ctx context.Context, opt ActivateOptions) error {
	if opt.ID == "" {
		return fmt.Errorf("empty command id")
	}
	if opt.StartFn == nil {
		return fmt.Errorf("StartFn is nil")
	}
	if opt.Timeout <= 0 {
		// 给个默认值，避免永不超时导致资源泄漏
		opt.Timeout = 30 * time.Second
	}

	st := &cmdState{
		id:       opt.ID,
		cmdErr:   make(chan error, 2), // 错误通道（缓冲区大小为1）
		finalErr: make(chan error, 1),
	}

	// 注册
	m.mu.Lock()
	if _, exists := m.cmds[opt.ID]; exists {
		m.mu.Unlock()
		return ErrCommandDuplicated
	}
	m.cmds[opt.ID] = st
	m.mu.Unlock()

	startErr := opt.StartFn(ctx, opt.ID)
	if startErr != nil {
		m.completeInternal(opt.ID, startErr)
		return startErr
	}

	//
	go func() {
		select {
		case <-ctx.Done():
			// canceled
			m.completeInternal(opt.ID, ctx.Err())
		case <-time.After(opt.Timeout):
			// timeout
			m.completeInternal(opt.ID, ErrCommandTimeout)
		case err := <-st.cmdErr:
			// completed
			m.completeInternal(opt.ID, err)
		}
	}()

	return nil
}

// CompleteCommand：由 RPC 层调用，标记命令完成（err==nil 表示成功）
func (m *CmdManager) CompleteCommand(commandID string, err error) error {
	m.mu.Lock()
	st, exists := m.cmds[commandID]
	m.mu.Unlock()

	if !exists {
		return ErrCommandNotFound
	}

	// 通过 cmdErr 错误通道传递错误信息
	st.cmdErr <- err
	return nil
}

// SyncMultipleCommands：返回一个 channel，依次写出每个 command 的 Result，全部完成后关闭 channel
func (m *CmdManager) SyncMultipleCommands(commandIDs ...string) <-chan Result {
	out := make(chan Result, len(commandIDs))

	var wg sync.WaitGroup
	wg.Add(len(commandIDs))

	for _, id := range commandIDs {
		id := id
		go func() {
			defer wg.Done()

			st := m.get(id)
			if st == nil {
				out <- Result{ID: id, Err: ErrCommandNotFound}
				return
			}

			// 等待命令完成，读取错误信息
			err := <-st.finalErr
			out <- Result{ID: id, Err: err}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

/* ----------------- internal helpers ----------------- */

func (m *CmdManager) get(id string) *cmdState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cmds[id]
}

func (m *CmdManager) completeInternal(id string, err error) {
	m.mu.Lock()
	st := m.cmds[id]
	m.mu.Unlock()
	if st == nil {
		return
	}

	st.once.Do(func() {

		st.finalErr <- err

		// cleanup map
		m.mu.Lock()
		delete(m.cmds, id)
		m.mu.Unlock()
	})
}

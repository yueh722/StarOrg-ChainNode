package logdb

import (
	"context"
	"errors"
	"io"
	"strconv"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Query struct {
	NodeID    string
	CommandID string
}

type CommandLogStore interface {
	Append(commandID, nodeID string, value []byte) error
	Scan(fn func(rec CommandLogRow) error, opt Query) error
	io.Closer
}

var (
	ErrClosed = errors.New("sqlitelogstore: closed")
)

// ===== GORM model =====

type CommandLogRow struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	Time      int64  `gorm:"not null;index:idx_time_id,priority:1"` // UnixMilli
	CommandID string `gorm:"type:text;not null;index:idx_cmd_node,priority:1"`
	NodeID    string `gorm:"type:text;not null;index:idx_cmd_node,priority:2"`
	Value     []byte `gorm:"type:blob;not null"`
}

func (CommandLogRow) TableName() string { return "command_logs" }

// ===== Store 实现（含 ctx 超时 + PRAGMA 初始化） =====

type Config struct {
	Path string
	Loc  *time.Location

	// InitTimeout：初始化（migrate + pragma）超时
	InitTimeout time.Duration
	// OpTimeout：每次 Append/Scan 的 DB 操作超时（Scan 会为“查询+逐行扫描”使用同一个 ctx）
	OpTimeout time.Duration

	// BusyTimeout：SQLite busy_timeout（毫秒）。0 表示不设置
	BusyTimeout time.Duration
}

type SqliteCommandLogStore struct {
	db    *gorm.DB
	rawDB *gorm.DB // ScanRows 用

	sqlDB  io.Closer // underlying *sql.DB
	loc    *time.Location
	closed bool

	opTimeout time.Duration
}

func NewSqliteCommandLogStore(cfg Config) (*SqliteCommandLogStore, error) {
	loc := cfg.Loc
	if loc == nil {
		loc = time.Local
	}
	initTO := cfg.InitTimeout
	if initTO <= 0 {
		initTO = 5 * time.Second
	}
	opTO := cfg.OpTimeout
	if opTO <= 0 {
		opTO = 3 * time.Second
	}

	gdb, err := gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}

	initCtx, cancel := context.WithTimeout(context.Background(), initTO)
	defer cancel()

	// 建表
	if err := gdb.WithContext(initCtx).AutoMigrate(&CommandLogRow{}); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	// WAL + synchronous
	if err := gdb.WithContext(initCtx).Exec(`PRAGMA journal_mode=WAL;`).Error; err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := gdb.WithContext(initCtx).Exec(`PRAGMA synchronous=NORMAL;`).Error; err != nil {
		_ = sqlDB.Close()
		return nil, err
	}

	// busy_timeout，“database is locked”
	if cfg.BusyTimeout > 0 {
		ms := int(cfg.BusyTimeout / time.Millisecond)
		if ms < 0 {
			ms = 0
		}
		if err := gdb.WithContext(initCtx).Exec(`PRAGMA busy_timeout=` + strconv.Itoa(ms) + `;`).Error; err != nil {
			_ = sqlDB.Close()
			return nil, err
		}
	}

	return &SqliteCommandLogStore{
		db:        gdb,
		rawDB:     gdb,
		sqlDB:     sqlDB,
		loc:       loc,
		opTimeout: opTO,
	}, nil
}

func (s *SqliteCommandLogStore) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.sqlDB != nil {
		return s.sqlDB.Close()
	}
	return nil
}

func (s *SqliteCommandLogStore) Append(commandID, nodeID string, value []byte) error {
	if s.closed {
		return ErrClosed
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.opTimeout)
	defer cancel()

	now := time.Now().In(s.loc)
	row := CommandLogRow{
		Time:      now.UnixMilli(),
		CommandID: commandID,
		NodeID:    nodeID,
		Value:     cloneBytes(value),
	}

	return s.db.WithContext(ctx).Create(&row).Error
}

// Scan：按“最近优先”扫描（time desc, id desc），并按 Query 做过滤。
// - opt.CommandID / opt.NodeID 为空则不作为过滤条件
// - 回调返回 error 会立即停止并返回该 error
// - DB 操作使用 ctx 超时（OpTimeout）
func (s *SqliteCommandLogStore) Scan(fn func(rec CommandLogRow) error, opt Query) error {
	if s.closed {
		return ErrClosed
	}
	if fn == nil {
		return errors.New("sqlitelogstore: nil scan func")
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.opTimeout)
	defer cancel()

	tx := s.db.WithContext(ctx).Model(&CommandLogRow{})
	if opt.NodeID != "" {
		tx = tx.Where("node_id = ?", opt.NodeID)
	}
	if opt.CommandID != "" {
		tx = tx.Where("command_id = ?", opt.CommandID)
	}
	tx = tx.Order("time desc").Order("id desc")

	rows, err := tx.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var r CommandLogRow
		if err := s.rawDB.WithContext(ctx).ScanRows(rows, &r); err != nil {
			return err
		}

		rec := CommandLogRow{
			Time:      r.Time,
			CommandID: r.CommandID,
			NodeID:    r.NodeID,
			Value:     cloneBytes(r.Value),
		}
		if err := fn(rec); err != nil {
			return err
		}
	}
	return nil
}

// ===== helpers =====

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	cp := make([]byte, len(b))
	copy(cp, b)
	return cp
}

package queuestore

import (
	"chain/utils"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type errorEventRow struct {
	EventID string `gorm:"column:event_id;primaryKey;size:128"`
	NodeID  string `gorm:"column:node_id;size:128;not null"`
	Seq     uint64 `gorm:"column:seq;not null;index:idx_sent_time_seq,priority:3"`
	TimeMS  int64  `gorm:"column:time_ms;not null;index:idx_sent_time_seq,priority:2"`
	Level   string `gorm:"column:level;size:16;not null"`
	JSON    string `gorm:"column:json;type:TEXT;not null"`
	Sent    bool   `gorm:"column:sent;not null;default:false;index:idx_sent_time_seq,priority:1"`
}

func toRow(e PersistedEvent) errorEventRow {
	return errorEventRow{
		EventID: e.EventID,
		NodeID:  e.NodeID,
		Seq:     e.Seq,
		TimeMS:  e.Time,
		Level:   e.Level,
		JSON:    e.JSON,
		Sent:    false,
	}
}

func fromRow(r errorEventRow) PersistedEvent {
	return PersistedEvent{
		EventID: r.EventID,
		NodeID:  r.NodeID,
		Seq:     r.Seq,
		Time:    r.TimeMS,
		Level:   r.Level,
		JSON:    r.JSON,
	}
}

func (errorEventRow) TableName() string { return "error_events" }

type SQLiteStore struct {
	db     *gorm.DB
	sqlDB  interface{ Close() error }
	opTO   time.Duration
	broker *Broker
}

type SQLiteConfig struct {
	Path        string
	Migrate     bool
	OpTimeout   time.Duration
	InitTimeout time.Duration
}

func NewSQLiteStore(cfg SQLiteConfig) (*SQLiteStore, error) {
	if cfg.Path == "" {
		return nil, errors.New("sqlite path is empty")
	}
	if cfg.OpTimeout <= 0 {
		cfg.OpTimeout = 2 * time.Second
	}
	if cfg.InitTimeout <= 0 {
		cfg.InitTimeout = 10 * time.Second
	}
	if !cfg.Migrate {
		cfg.Migrate = true
	}

	conn, err := utils.SqliteOpen(cfg.Path)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(conn, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	initCtx, cancel := context.WithTimeout(context.Background(), cfg.InitTimeout)
	defer cancel()

	// PRAGMA（可选）
	if err := db.WithContext(initCtx).Exec(`PRAGMA journal_mode=WAL;`).Error; err != nil {
		return nil, err
	}
	if err := db.WithContext(initCtx).Exec(`PRAGMA synchronous=NORMAL;`).Error; err != nil {
		return nil, err
	}

	if cfg.Migrate {
		if err := db.WithContext(initCtx).AutoMigrate(&errorEventRow{}); err != nil {
			return nil, err
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	return &SQLiteStore{
		db:     db,
		sqlDB:  sqlDB,
		opTO:   cfg.OpTimeout,
		broker: NewBroker(),
	}, nil
}

func (s *SQLiteStore) GetBroker() *Broker {
	return s.broker
}

func (s *SQLiteStore) Close() error {
	if s.sqlDB != nil {
		return s.sqlDB.Close()
	}
	return nil
}

func (s *SQLiteStore) Put(e PersistedEvent) error {
	if e.EventID == "" {
		return ErrEmptyEventID
	}

	s.broker.Publish(e)

	ctx, cancel := context.WithTimeout(context.Background(), s.opTO)
	defer cancel()

	row := toRow(e)
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "event_id"}},
			DoNothing: true,
		}).
		Create(&row).Error
}

func (s *SQLiteStore) MarkSent(eventID string) error {
	if eventID == "" {
		return ErrEmptyEventID
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.opTO)
	defer cancel()

	return s.db.WithContext(ctx).
		Model(&errorEventRow{}).
		Where("event_id = ?", eventID).
		Update("sent", true).Error
}

func (s *SQLiteStore) ScanUnsent(limit int, fn func(PersistedEvent) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	q := s.db.WithContext(ctx).
		Model(&errorEventRow{}).
		Where("sent = ?", false).
		Order("time_ms ASC, seq ASC")

	if limit > 0 {
		q = q.Limit(limit)
	}

	rows, err := q.Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var r errorEventRow
		if err := s.db.ScanRows(rows, &r); err != nil {
			return err
		}
		if err := fn(fromRow(r)); err != nil {
			return err
		}
	}
	return nil
}

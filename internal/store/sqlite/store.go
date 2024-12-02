package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/kavehrafie/go-scheduler/internal/model"
	"github.com/kavehrafie/go-scheduler/internal/store"
	"github.com/kavehrafie/go-scheduler/pkg/database"
	"log"
	_ "modernc.org/sqlite"
	"time"
)

type sqliteStore struct {
	db *sql.DB
}

type SQLiteFactory struct {
}

func (f *SQLiteFactory) NewStore(config database.Config) (store.Store, error) {
	if config.Driver != database.SQLite {
		return nil, store.ErrInvalidDriver
	}

	db, err := sql.Open("sqlite", config.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Enable foreign keys and WAL mode for better performance
	if _, err := db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set PRAGMA: %w", err)
	}

	// Set some reasonable defaults
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	s := &sqliteStore{db: db}
	log.Println("Connected to SQLite database")
	if err := s.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}

	return s, nil
}

func (s *sqliteStore) Query(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled requests: %w", err)
	}

	return rows, nil
}

func (s *sqliteStore) Exec(query string, args ...interface{}) error {
	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute query: %w", err)
	}

	return nil
}

func (s *sqliteStore) initSchema() error {

	for _, schema := range tables() {
		_, err := s.db.Exec(schema)
		if err != nil {
			return fmt.Errorf("failed to create schema: %w", err)
		}

	}

	return nil
}

func (s *sqliteStore) Create(ctx context.Context, sa *model.ScheduledRequest) error {
	metadata, err := json.Marshal(sa.Header)
	if err != nil {
		return fmt.Errorf("failed to marshal scheduled action metadata: %v", err)
	}

	query := `
	INSERT INTO scheduled_actions (
                               id, title, status, description, url, payload, metadata, scheduled_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query,
		sa.ID,
		sa.Title,
		model.StatusPending,
		sa.Description,
		sa.URL,
		sa.Payload,
		metadata,
		sa.ScheduledAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create scheduled action: %v", err)
	}
	if rows, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	} else if rows == 0 {
		return store.ErrAlreadyExists
	}

	return nil
}

//
//func (s *sqliteStore) Get(ctx context.Context, id string) (*model.ScheduledAction, error) {
//	query := `SELECT * FROM scheduled_actions WHERE id = ?`
//
//	var (
//		sa       model.ScheduledAction
//		metadata []byte
//	)
//
//	err := s.db.QueryRowContext(ctx, query, id).Scan(
//		&sa.ID,
//		&sa.Title,
//		&sa.Status,
//		&sa.Description,
//		&sa.Payload,
//		&sa.ScheduledAt,
//		&metadata,
//		&sa.Failures,
//	)
//
//	if errors.Is(err, sql.ErrNoRows) {
//		return nil, store.ErrNotFound
//	}
//	if err != nil {
//		return nil, fmt.Errorf("failed to get the scheduled action: %v", err)
//	}
//
//	if err := json.Unmarshal(metadata, &sa.Metadata); err != nil {
//		return nil, fmt.Errorf("failed to unmarshal scheduled action metadata: %v", err)
//	}
//
//	return &sa, nil
//}
//
//func (s *sqliteStore) Update(ctx context.Context, sa *model.ScheduledAction) error {
//	metadata, err := json.Marshal(sa.Metadata)
//	if err != nil {
//		return fmt.Errorf("failed to marshal scheduled action metadata: %v", err)
//	}
//
//	query := `
//	UPDATE scheduled_actions
//	SET title = ?, description = ?,  metadata = ?, scheduled_at = ? failures = ?, updated_at = ?
//	WHERE id = ?
//`
//	result, err := s.db.ExecContext(ctx, query,
//		sa.Title,
//		sa.Description,
//		metadata,
//		sa.ScheduledAt,
//		sa.Failures,
//		time.Now(),
//		sa.ID,
//	)
//	if err != nil {
//		return fmt.Errorf("failed to update scheduled action: %v", err)
//	}
//
//	rows, err := result.RowsAffected()
//	if err != nil {
//		return fmt.Errorf("failed to get rows affected: %v", err)
//	}
//	if rows == 0 {
//		return store.ErrNotFound
//	}
//
//	return nil
//}
//
//func (s *sqliteStore) Delete(ctx context.Context, id string) error {
//	result, err := s.db.ExecContext(ctx, `DELETE FROM scheduled_actions WHERE id = ?`, id)
//	if err != nil {
//		return fmt.Errorf("failed to delete scheduled action: %v", err)
//	}
//
//	rows, err := result.RowsAffected()
//	if err != nil {
//		return fmt.Errorf("failed to get rows affected: %v", err)
//	}
//	if rows == 0 {
//		return store.ErrNotFound
//	}
//	return nil
//}
//
//func (s *sqliteStore) List(ctx context.Context, offset, limit int) ([]*model.ScheduledAction, error) {
//	query := `SELECT * FROM scheduled_actions LIMIT ? OFFSET ?`
//
//	return s.queryActions(ctx, query, offset, limit)
//}
//
//func (s *sqliteStore) ListByStatus(ctx context.Context, status model.ScheduledActionStatus) ([]*model.ScheduledAction, error) {
//	query := `SELECT * FROM scheduled_actions WHERE status = ? ORDER BY scheduled_at DESC`
//
//	return s.queryActions(ctx, query, status)
//}
//
//func (s *sqliteStore) ListPending(ctx context.Context, before time.Time) ([]*model.ScheduledAction, error) {
//	query := `SELECT * FROM scheduled_actions WHERE status = ? AND scheduled_at <= ? ORDER BY scheduled_at ASC`
//
//	return s.queryActions(ctx, query, model.StatusPending, before)
//}
//
//func (s *sqliteStore) queryActions(ctx context.Context, query string, arg ...interface{}) ([]*model.ScheduledAction, error) {
//	rows, err := s.db.QueryContext(ctx, query, arg...)
//	if err != nil {
//		return nil, fmt.Errorf("failed to query scheduled actions: %v", err)
//	}
//	defer rows.Close()
//
//	var sas []*model.ScheduledAction
//	for rows.Next() {
//		var (
//			sa       model.ScheduledAction
//			metadata []byte
//		)
//
//		err := rows.Scan(
//			&sa.ID,
//			&sa.Title,
//			&sa.Status,
//			&sa.Description,
//			&sa.URL,
//			&sa.Payload,
//			&metadata,
//			&sa.CreatedAt,
//			&sa.ScheduledAt,
//			&sa.Failures,
//			&sa.UpdatedAt,
//			&sa.DeletedAt,
//		)
//
//		if err != nil {
//			return nil, fmt.Errorf("failed to scan scheduled action: %v", err)
//		}
//		if err := json.Unmarshal(metadata, &sa.Metadata); err != nil {
//			return nil, fmt.Errorf("failed to unmarshal scheduled action metadata: %v", err)
//		}
//
//		sas = append(sas, &sa)
//	}
//	if err := rows.Err(); err != nil {
//		return nil, fmt.Errorf("failed to iterate over scheduled actions: %v", err)
//	}
//
//	return sas, nil
//}

func (s *sqliteStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}

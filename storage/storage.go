// Package storage отвечает только за базу данных (SQLite).
// О gRPC и игре этот пакет ничего не знает.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Storage struct {
	db *sql.DB
}

// Entry - одна строка рейтинга.
type Entry struct {
	Player    string
	Score     int32
	UpdatedAt int64
}

// Open открывает (или создаёт) базу и таблицу.
func Open(path string) (*Storage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite лучше работает с одним соединением

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS players (
		name       TEXT PRIMARY KEY,
		best       INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("создание таблицы: %w", err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

// SubmitScore сохраняет результат (в базе остаётся лучший счёт игрока)
// и возвращает лучший счёт и место игрока в рейтинге.
func (s *Storage) SubmitScore(ctx context.Context, name string, score int32) (best, rank int32, err error) {
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO players(name, best, updated_at) VALUES(?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			best = MAX(best, excluded.best),
			updated_at = CASE WHEN excluded.best > best THEN excluded.updated_at ELSE updated_at END`,
		name, score, time.Now().Unix())
	if err != nil {
		return 0, 0, err
	}

	err = s.db.QueryRowContext(ctx, `SELECT best FROM players WHERE name = ?`, name).Scan(&best)
	if err != nil {
		return 0, 0, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) + 1 FROM players WHERE best > ?`, best).Scan(&rank)
	if err != nil {
		return 0, 0, err
	}
	return best, rank, nil
}

// Top возвращает лучших игроков.
func (s *Storage) Top(ctx context.Context, limit int) ([]Entry, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT name, best, updated_at FROM players ORDER BY best DESC, updated_at ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Entry
	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.Player, &e.Score, &e.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

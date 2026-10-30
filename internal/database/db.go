// Package database ouvre la base SQLite de l'instance et expose les requêtes
// générées par sqlc (research.md › R2).
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/balsakup/frigg/internal/database/sqlc"

	_ "modernc.org/sqlite" // pilote SQLite en Go pur
)

// Store regroupe les deux pools de connexions : un seul écrivain pour éviter
// « database is locked », plusieurs lecteurs en lecture seule.
type Store struct {
	ReadDB  *sql.DB
	WriteDB *sql.DB
	Read    *sqlc.Queries
	Write   *sqlc.Queries
}

// Open ouvre (et crée au besoin) la base située à path.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, fmt.Errorf("création du répertoire de la base : %w", err)
		}
	}

	writeDB, err := sql.Open("sqlite", dsn(path, false))
	if err != nil {
		return nil, fmt.Errorf("ouverture de la base : %w", err)
	}
	writeDB.SetMaxOpenConns(1)

	readDB, err := sql.Open("sqlite", dsn(path, true))
	if err != nil {
		writeDB.Close()
		return nil, fmt.Errorf("ouverture de la base : %w", err)
	}
	readDB.SetMaxOpenConns(4)

	// Le pool d'écriture crée le fichier et active le mode WAL avant toute lecture.
	if err := writeDB.Ping(); err != nil {
		writeDB.Close()
		readDB.Close()
		return nil, fmt.Errorf("connexion à la base : %w", err)
	}

	return &Store{
		ReadDB:  readDB,
		WriteDB: writeDB,
		Read:    sqlc.New(readDB),
		Write:   sqlc.New(writeDB),
	}, nil
}

func dsn(path string, readOnly bool) string {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "synchronous(NORMAL)")
	if readOnly {
		q.Add("_pragma", "query_only(1)")
	} else {
		// Toute transaction démarre en BEGIN IMMEDIATE.
		q.Set("_txlock", "immediate")
	}
	return "file:" + path + "?" + q.Encode()
}

// WriteTx exécute fn dans une transaction d'écriture, validée si fn ne renvoie
// pas d'erreur.
func (s *Store) WriteTx(ctx context.Context, fn func(q *sqlc.Queries) error) error {
	tx, err := s.WriteDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("début de transaction : %w", err)
	}
	if err := fn(s.Write.WithTx(tx)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("validation de la transaction : %w", err)
	}
	return nil
}

// Close ferme les deux pools.
func (s *Store) Close() error {
	return errors.Join(s.ReadDB.Close(), s.WriteDB.Close())
}

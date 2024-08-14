package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

const PPACER_FF_ENV_LOG_LEVEL = "PPACER_FF_LOG_LEVEL"

var (
	ErrUserNotFound  = errors.New("user not found in database")
	ErrUserNotUnique = errors.New("more than single user with given email")
)

type UserRow struct {
	Email          string
	Nickname       *string
	Hash           string
	RegistrationTs string
	Confirmed      int
	ConfirmationTs string
}

func UserByEmail(db *SqliteDB, email string) (UserRow, error) {
	rows, qErr := db.Query(readUserByEmailQuery(), email)
	if qErr != nil {
		return UserRow{}, fmt.Errorf("cannot query user by email: %w", qErr)
	}
	defer rows.Close()
	rowsN := 0
	var userRow UserRow
	var scanErr error

	for rows.Next() {
		rowsN += 1
		userRow, scanErr = parseUserRow(rows)
		if scanErr != nil {
			return userRow, fmt.Errorf("error while scanning userRow: %w",
				scanErr)
		}
	}
	if rowsN == 0 {
		return UserRow{}, ErrUserNotFound
	}
	if rowsN != 1 {
		return UserRow{}, ErrUserNotUnique
	}
	return userRow, nil
}

func UserByHash(db *SqliteDB, hash string) (UserRow, error) {
	rows, qErr := db.Query(readUserByHashQuery(), hash)
	if qErr != nil {
		return UserRow{}, fmt.Errorf("cannot query user by hash: %w", qErr)
	}
	defer rows.Close()
	rowsN := 0
	var userRow UserRow
	var scanErr error

	for rows.Next() {
		rowsN += 1
		userRow, scanErr = parseUserRow(rows)
		if scanErr != nil {
			return userRow, fmt.Errorf("error while scanning userRow: %w",
				scanErr)
		}
	}
	if rowsN == 0 {
		return UserRow{}, ErrUserNotFound
	}
	if rowsN != 1 {
		return UserRow{}, ErrUserNotUnique
	}
	return userRow, nil
}

func InsertNewUser(db *SqliteDB, user User) error {
	confirmed := 0
	if user.Confirmed {
		confirmed = 1
	}
	_, iErr := db.Exec(
		insertNewUserQuery(),
		user.Email, user.Nickname, user.Hash, ToString(user.RegistrationTs),
		confirmed, ToString(user.ConfirmationTs),
	)
	if iErr != nil {
		return iErr
	}
	return nil
}

func ConfirmUser(db *SqliteDB, email, hash string) error {
	now := ToString(time.Now())
	stats, iErr := db.Exec(confirmUserQuery(), now, email, hash)
	if iErr != nil {
		return iErr
	}
	rows, rErr := stats.RowsAffected()
	if rErr != nil {
		return fmt.Errorf("cannot get number of rows affected: %w", rErr)
	}
	if rows != 1 {
		return fmt.Errorf("updated more than single user for email=%s and hash=%s: %d",
			email, hash, rows)
	}
	return nil
}

func parseUserRow(rows *sql.Rows) (UserRow, error) {
	var email, hash, regTs, confTs string
	var nickname *string
	var confirmed int
	scanErr := rows.Scan(&email, &nickname, &hash, &regTs, &confirmed, &confTs)
	if scanErr != nil {
		return UserRow{}, scanErr
	}
	userRow := UserRow{
		Email:          email,
		Nickname:       nickname,
		Hash:           hash,
		RegistrationTs: regTs,
		Confirmed:      confirmed,
		ConfirmationTs: confTs,
	}
	return userRow, nil
}

func readUserByEmailQuery() string {
	return `
	SELECT
		Email,
		Nickname,
		Hash,
		RegistrationTs,
		Confirmed,
		ConfirmationTs
	FROM
		users
	WHERE
		Email = ?
`
}

func readUserByHashQuery() string {
	return `
	SELECT
		Email,
		Nickname,
		Hash,
		RegistrationTs,
		Confirmed,
		ConfirmationTs
	FROM
		users
	WHERE
		Hash = ?
`
}

func insertNewUserQuery() string {
	return `
	INSERT INTO users(Email, Nickname, Hash, RegistrationTs, Confirmed, ConfirmationTs)
	VALUES (?,?,?,?,?,?)
	`
}

func confirmUserQuery() string {
	return `
	UPDATE
		users
	SET
		Confirmed = 1,
		ConfirmationTs = ?
	WHERE
			Email = ?
		AND Hash = ?
`
}

func NewSqliteClient(dbFilePath string, logger *slog.Logger) (*SqliteDB, error) {
	if logger == nil {
		logger = defaultLogger()
	}
	sqliteDb, err := newSqliteClientForSchema(
		dbFilePath, logger, setupSqliteSchema,
	)
	if err != nil {
		return nil, err
	}
	return sqliteDb, nil
}

func newSqliteClientForSchema(
	dbFilePath string, logger *slog.Logger, setupSchemaFunc func(*sql.DB) error,
) (*SqliteDB, error) {
	dbFilePathAbs, absErr := filepath.Abs(dbFilePath)
	if absErr != nil {
		return nil, fmt.Errorf("cannot get absolute path of database file %s: %w",
			dbFilePath, absErr)
	}
	newDbCreated, dbFileErr := createSqliteDbIfNotExist(dbFilePathAbs)
	if dbFileErr != nil {
		return nil, fmt.Errorf("cannot create new empty SQLite database: %w",
			dbFileErr)
	}
	connString := sqliteConnString(dbFilePathAbs)
	db, dbErr := sql.Open("sqlite", connString)
	if dbErr != nil {
		return nil, fmt.Errorf("cannot connect to SQLite DB (%s): %w",
			connString, dbErr)
	}
	if newDbCreated {
		schemaErr := setupSchemaFunc(db)
		if schemaErr != nil {
			db.Close()
			return nil, fmt.Errorf("cannot setup SQLite schema for %s: %w",
				connString, schemaErr)
		}
	}
	return &SqliteDB{dbConn: db, dbFilePath: dbFilePathAbs}, nil
}

func sqliteConnString(dbFilePath string) string {
	// TODO: probably read from the config not only database file path but also
	// additional arguments also.
	options := "cache=shared&mode=rwc&_journal_mode=WAL"
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("%s?%s", dbFilePath, options)
	}
	return fmt.Sprintf("file://%s?%s", dbFilePath, options)
}

func setupSqliteSchema(db *sql.DB) error {
	schemaStmts, err := schemaStatements("sqlite")
	if err != nil {
		return err
	}
	return execSqlStatements(db, schemaStmts)
}

type SqliteDB struct {
	sync.RWMutex
	dbConn     *sql.DB
	dbFilePath string
	logger     *slog.Logger
}

func (s *SqliteDB) Begin() (*sql.Tx, error) {
	s.Lock()
	defer s.Unlock()
	return s.dbConn.Begin()
}

func (s *SqliteDB) Exec(query string, args ...any) (sql.Result, error) {
	s.Lock()
	defer s.Unlock()
	return s.dbConn.Exec(query, args...)
}

func (s *SqliteDB) ExecContext(
	ctx context.Context, query string, args ...any,
) (sql.Result, error) {
	s.Lock()
	defer s.Unlock()
	return s.dbConn.ExecContext(ctx, query, args...)
}

func (s *SqliteDB) Close() error {
	s.Lock()
	defer s.Unlock()
	return s.dbConn.Close()
}

func (s *SqliteDB) DataSource() string {
	return s.dbFilePath
}

func (s *SqliteDB) Query(query string, args ...any) (*sql.Rows, error) {
	s.RLock()
	defer s.RUnlock()
	return s.dbConn.Query(query, args...)
}

func (s *SqliteDB) QueryContext(
	ctx context.Context, query string, args ...any,
) (*sql.Rows, error) {
	s.RLock()
	defer s.RUnlock()
	return s.dbConn.QueryContext(ctx, query, args...)
}

func (s *SqliteDB) QueryRow(query string, args ...any) *sql.Row {
	s.RLock()
	defer s.RUnlock()
	return s.dbConn.QueryRow(query, args...)
}

func (s *SqliteDB) QueryRowContext(
	ctx context.Context, query string, args ...any,
) *sql.Row {
	s.RLock()
	defer s.RUnlock()
	return s.dbConn.QueryRowContext(ctx, query, args...)
}

func execSqlStatements(db *sql.DB, stmts []string) error {
	for _, query := range stmts {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		_, err := db.Exec(query)
		if err != nil {
			return err
		}
	}
	return nil
}

func createSqliteDbIfNotExist(dbFilePath string) (bool, error) {
	if _, err := os.Stat(dbFilePath); os.IsNotExist(err) {
		dirErr := os.MkdirAll(filepath.Dir(dbFilePath), os.ModePerm)
		if dirErr != nil {
			return false, dirErr
		}

		file, fErr := os.Create(dbFilePath)
		if fErr != nil {
			return false, fErr
		}
		file.Close()
		return true, nil
	}

	return false, nil
}

func schemaStatements(dbDriver string) ([]string, error) {
	if dbDriver == "sqlite" || dbDriver == "sqlite3" {
		return []string{
			sqliteSetupWAL(),
			sqliteCreateUserTable(),
		}, nil
	}

	return []string{}, fmt.Errorf("there is no schema for %s driver defined",
		dbDriver)
}

func sqliteSetupWAL() string {
	return "PRAGMA journal_mode = WAL;"
}

func sqliteCreateUserTable() string {
	return `
		CREATE TABLE IF NOT EXISTS users (
			Email          TEXT NOT NULL,
			Nickname       TEXT NULL,
			Hash           TEXT NOT NULL,
			RegistrationTs TEXT NOT NULL,
			Confirmed      INT NOT NULL,
			ConfirmationTs TEXT NOT NULL,

			PRIMARY KEY (Email)
		);
`
}

func defaultLogger() *slog.Logger {
	level := os.Getenv(PPACER_FF_ENV_LOG_LEVEL)
	var logLevel slog.Level
	switch level {
	case "DEBUG":
		logLevel = slog.LevelDebug
	case "INFO":
		logLevel = slog.LevelInfo
	case "WARN":
		logLevel = slog.LevelWarn
	case "ERROR":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelWarn
	}
	opts := slog.HandlerOptions{Level: logLevel}
	return slog.New(slog.NewTextHandler(os.Stdout, &opts))
}

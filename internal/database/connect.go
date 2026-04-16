package database

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/unkmonster/tmd/internal/utils"
	log "github.com/sirupsen/logrus"
)

func Connect(path string) (*sqlx.DB, error) {
	ex, err := utils.PathExists(path)
	if err != nil {
		return nil, fmt.Errorf("failed to check if db file exists at %q: %w", path, err)
	}

	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&busy_timeout=2147483647", path)
	db, err := sqlx.Connect("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database at %q: %w", path, err)
	}

	CreateTables(db)
	if err := MigrateDatabase(db); err != nil {
		return nil, fmt.Errorf("failed to migrate database at %q: %w", path, err)
	}

	if !ex {
		log.Debugln("created new db file", path)
	}
	return db, nil
}

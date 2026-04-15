package database

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func CreateLst(db *sqlx.DB, lst *Lst) error {
	stmt := `INSERT INTO lsts(id, name, owner_uid) VALUES(:id, :name, :owner_uid)`
	_, err := db.NamedExec(stmt, &lst)
	if err != nil {
		return fmt.Errorf("failed to create list %d (%s): %w", lst.Id, lst.Name, err)
	}
	return nil
}

func DelLst(db *sqlx.DB, lid uint64) error {
	stmt := `DELETE FROM lsts WHERE id=?`
	_, err := db.Exec(stmt, lid)
	if err != nil {
		return fmt.Errorf("failed to delete list %d: %w", lid, err)
	}
	return nil
}

func GetLst(db *sqlx.DB, lid uint64) (*Lst, error) {
	stmt := `SELECT * FROM lsts WHERE id = ?`
	result := &Lst{}
	err := db.Get(result, stmt, lid)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get list %d: %w", lid, err)
	}
	return handleGetResult(result, err)
}

func UpdateLst(db *sqlx.DB, lst *Lst) error {
	stmt := `UPDATE lsts SET name=? WHERE id=?`
	_, err := db.Exec(stmt, lst.Name, lst.Id)
	if err != nil {
		return fmt.Errorf("failed to update list %d: %w", lst.Id, err)
	}
	return nil
}

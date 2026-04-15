package database

import (
	"database/sql"
)

func handleGetResult[T any](result *T, err error) (*T, error) {
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func handleInsertWithId(res sql.Result, err error, idScanner func(int64)) error {
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	idScanner(id)
	return nil
}

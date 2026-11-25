package repository

import (
	"database/sql"
	"errors"
)

func isSQLNoRowsError(err error) bool { return errors.Is(err, sql.ErrNoRows) }

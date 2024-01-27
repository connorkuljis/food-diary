package repo

import (
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var db *sqlx.DB

const DbName = "meals.db"

func InitDB() error {
	var err error

	db, err = sqlx.Connect("sqlite", DbName)
	if err != nil {
		return err
	}

	_, err = db.Exec(mealsSchema)
	if err != nil {
		return err
	}
	return nil
}

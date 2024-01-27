package repo

import (
	"log"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var db *sqlx.DB

const DbName = ".food-diary/meals.db"

func InitDB() error {
	var err error

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	db, err = sqlx.Connect("sqlite", filepath.Join(home, DbName))
	if err != nil {
		return err
	}

	_, err = db.Exec(mealsSchema)
	if err != nil {
		return err
	}
	return nil
}

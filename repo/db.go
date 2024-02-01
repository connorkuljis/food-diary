package repo

import (
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var db *sqlx.DB

const DbName = ".meals.db"

func InitDB() error {
	var err error

	// home, err := os.UserHomeDir()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// db, err = sqlx.Connect("sqlite", filepath.Join(home, DbName))
	db, err = sqlx.Connect("sqlite", DbName)
	if err != nil {
		return err
	}

	_, err = db.Exec(MealsSchema)
	if err != nil {
		return err
	}

	_, err = db.Exec(UsersSchema)
	if err != nil {
		return err
	}
	return nil
}

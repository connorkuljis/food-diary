package repo

import (
	"time"

	_ "github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

type Meal struct {
	Id           int64  `db:"id"`
	UserID       int64  `db:"user_id"`
	Name         string `db:"name"`
	MealType     string `db:"meal_type"`
	DateConsumed string `db:"date_consumed"`
}

var MealsSchema = `CREATE TABLE IF NOT EXISTS Meals (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER REFERENCES Users(id),
	name TEXT NOT NULL,
	meal_type TEXT NOT NULL,
	date_consumed TEXT NOT NULL
)`

type MealType string

const (
	Timestamp = "2006-01-02 15:04:05"

	Breakfast MealType = "breakfast"
	Lunch     MealType = "lunch"
	Dinner    MealType = "dinner"
	Snacks    MealType = "snacks"
)

func NewMeal(name string, userId int64, mealType MealType, time time.Time) Meal {
	return Meal{
		Name:         name,
		UserID:       userId,
		MealType:     string(mealType),
		DateConsumed: time.Format(Timestamp),
	}
}

func InsertMeal(meal Meal) (Meal, error) {
	query := `INSERT INTO Meals(name, user_id, meal_type, date_consumed) VALUES (:name, :user_id, :meal_type, :date_consumed)`

	res, err := db.NamedExec(query, meal)
	if err != nil {
		return meal, err
	}

	lastInsertID, err := res.LastInsertId()
	if err != nil {
		return meal, err
	}

	meal.Id = lastInsertID

	return meal, nil
}

func GetAllMeals() ([]Meal, error) {
	query := `SELECT * FROM Meals`

	var meals []Meal
	err := db.Select(&meals, query)
	if err != nil {
		return meals, err
	}

	return meals, nil
}

func GetMealsByUserAndDate(user User, inTime time.Time) ([]Meal, error) {
	query := `SELECT * FROM Meals WHERE user_id = ? AND DATE(date_consumed) = DATE(?)`

	var meals []Meal

	err := db.Select(&meals, query, user.Id, inTime.Format("2006-01-02"))
	if err != nil {
		return meals, err
	}

	return meals, nil
}

func DeleteMealByUserAndId(user User, id string) error {
	query := `DELETE FROM Meals WHERE user_id = ? AND id = ?`

	res, err := db.Exec(query, user.Id, id)
	if err != nil {
		return err
	}

	_, err = res.LastInsertId()
	if err != nil {
		return err
	}

	return nil
}

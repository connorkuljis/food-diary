package repo

import (
	"time"

	_ "github.com/jmoiron/sqlx"
)

type Meal struct {
	Id           int64     `db:"id"`
	Name         string    `db:"name"`
	MealType     string    `db:"meal_type"`
	DateConsumed time.Time `db:"date_consumed"`
}

var mealsSchema = `CREATE TABLE IF NOT EXISTS Meals (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	meal_type TEXT NOT NULL,
	date_consumed TIMESTAMP NOT NULL
)`

type MealType string

const (
	Breakfast MealType = "breakfast"
	Lunch     MealType = "lunch"
	Dinner    MealType = "dinner"
	Snacks    MealType = "snacks"
)

func NewMeal(name string, mealType MealType, dateConsumed time.Time) Meal {
	return Meal{
		Name:         name,
		MealType:     string(mealType),
		DateConsumed: dateConsumed,
	}
}

func InsertMeal(meal Meal) (Meal, error) {
	query := `INSERT INTO Meals(name, meal_type, date_consumed) VALUES (:name, :meal_type, :date_consumed)`

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

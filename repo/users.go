package repo

type User struct {
	Id       int64  `db:"id"`
	Email    string `db:"email"`
	Password string `db:"password"`
}

var UsersSchema = `CREATE TABLE IF NOT EXISTS Users(
	id INTEGER PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	password TEXT NOT NULL
	)`

func NewUser(email, password string) User {
	return User{
		Email:    email,
		Password: password,
	}
}

func InsertUser(user User) (User, error) {
	query := "INSERT INTO Users (email, password) VALUES (:email, :password)"

	res, err := db.NamedExec(query, user)
	if err != nil {
		return user, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return user, err
	}

	user.Id = id

	return user, nil
}

func GetUserByEmail(email string) (User, error) {
	query := "SELECT * FROM Users WHERE email = ?"

	var user User
	err := db.Get(&user, query, email)
	if err != nil {
		return user, err
	}

	return user, nil
}

package server

import (
	"errors"
	"net/http"

	"github.com/gorilla/sessions"
)

func GetUserId(r *http.Request, s *sessions.CookieStore) (int64, error) {
	const key = "userId"

	session, _ := s.Get(r, "session")

	id, ok := session.Values[key].(int64)
	if !ok {
		return 0, errors.New("Error! Could not get user id from session")
	}

	return id, nil
}

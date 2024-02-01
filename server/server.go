package server

import (
	"database/sql"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/connorkuljis/food-diary/repo"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"modernc.org/sqlite"
)

// Server encapsulates all dependencies for the web server.
// HTTP handlers access information via receiver types.
type Server struct {
	FileSystem fs.FS // in-memory or disk
	Router     *chi.Mux
	Sessions   *sessions.CookieStore
	SiteData   SiteData

	Port         string
	StaticDir    string // location of static assets
	TemplatesDir string // location of html templates, makes template parsing less verbose.
}

type SiteData struct {
	Title string
}

const (
	Port             = "8081"
	StaticDirName    = "/static"
	TemplatesDirName = "/templates"
)

func NewServer(fs fs.FS) *Server {
	router := chi.NewMux()
	store := sessions.NewCookieStore([]byte("3lWcaN9nYFjh9Dy5RJWXR84nxYSOZSQx4R11y8NxUNQ="))
	siteData := SiteData{Title: "Food Diary"}

	return &Server{
		FileSystem:   fs,
		Router:       router,
		Sessions:     store,
		Port:         Port,
		StaticDir:    StaticDirName,
		TemplatesDir: TemplatesDirName,
		SiteData:     siteData,
	}
}

// This function automatically builds *template.Templates using filenames and
func (s *Server) CompileTemplates(name string, files []HTMLFile, funcMap template.FuncMap) *template.Template {
	// give the template a name
	tmpl := template.New(name)

	// add an optional funcmap
	if funcMap != nil {
		tmpl.Funcs(funcMap)
	}

	// build up the array of filenames needed to parse on the fs
	var patterns []string
	for _, file := range files {
		patterns = append(patterns, string(file))
	}

	// build the template
	tmpl, err := tmpl.ParseFS(s.FileSystem, patterns...)
	if err != nil {
		log.Fatal(err)
	}

	return tmpl
}

func (s *Server) Routes() {
	s.Router.Handle("/static/*", http.FileServer(http.FS(s.FileSystem)))
	s.Router.HandleFunc("/", s.handleIndex())

	// Template rendering
	s.Router.HandleFunc("/today", s.handleToday(TodayView))
	s.Router.HandleFunc("/login", s.handleLogin(LoginView))
	s.Router.HandleFunc("/register", s.handleRegister(RegisterView))
	s.Router.HandleFunc("/history", s.handleHistory(HistoryView))

	// HTMX 'n AJAX
	s.Router.HandleFunc("/logout", s.handleLogout())
	s.Router.Post("/api/meals", s.handleMeals())
	s.Router.Delete("/api/meals/{id}", s.handleDeleteMeal())
}

func ServerError(w http.ResponseWriter, err error) {
	log.Print(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func (s *Server) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/today", http.StatusSeeOther)
	}
}

func (s *Server) handleToday(view []HTMLFile) http.HandlerFunc {
	type ViewData struct {
		SiteData SiteData
		Meals    []repo.Meal
	}

	tmpl := s.CompileTemplates("today.html", view, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		// get the user id from the cookie
		// if user id not found in cookie, they are send to the login page
		userId, err := GetUserId(r, s.Sessions)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		meals, err := repo.GetMealsByUserAndDate(repo.User{Id: userId}, time.Now())
		if err != nil {
			ServerError(w, err)
			return
		}

		tmpl.ExecuteTemplate(w, "root", ViewData{
			SiteData: s.SiteData,
			Meals:    meals,
		})
	}
}

func (s *Server) handleLogin(view []HTMLFile) http.HandlerFunc {
	type ViewData struct {
		SiteData     SiteData
		ErrorMessage string
	}

	tmpl := s.CompileTemplates("login.html", view, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		data := ViewData{
			SiteData: s.SiteData,
		}
		data.SiteData.Title = data.SiteData.Title + " | Login"

		if r.Method == "GET" {
			tmpl.ExecuteTemplate(w, "root", data)
		}

		if r.Method == "POST" {
			// generate a cookie
			session, _ := s.Sessions.Get(r, "session")

			// handle the form
			r.ParseForm()
			emailStr := r.Form.Get("email")
			passwordStr := r.Form.Get("password")

			// look up the user by email
			user, err := repo.GetUserByEmail(emailStr)
			if err != nil {
				log.Print(err)
				data.ErrorMessage = "Invalid email or password"
				tmpl.ExecuteTemplate(w, "root", data)
				return
			}

			// compare the hashed passwords
			err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(passwordStr))
			if err != nil {
				log.Print(err)
				data.ErrorMessage = "Invalid email or password"
				tmpl.ExecuteTemplate(w, "root", data)
				return
			}

			// save the user id to the cookie
			session.Values["userId"] = user.Id
			err = sessions.Save(r, w)
			if err != nil {
				ServerError(w, err)
				return
			}

			// send the user to today
			http.Redirect(w, r, "/today", http.StatusSeeOther)
		}
	}
}

func (s *Server) handleRegister(view []HTMLFile) http.HandlerFunc {
	type ViewData struct {
		SiteData     SiteData
		ErrorMessage string
	}

	tmpl := s.CompileTemplates("register.html", view, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		data := ViewData{SiteData: s.SiteData}

		// update the site title
		data.SiteData.Title += " | Register"

		if r.Method == "GET" {
			tmpl.ExecuteTemplate(w, "root", data)
		}

		if r.Method == "POST" {
			// generate a cookie
			session, _ := s.Sessions.Get(r, "session")

			// handle the form
			r.ParseForm()
			emailStr := r.Form.Get("email")
			passwordStr := r.Form.Get("password")

			// hash the password
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordStr), 10)
			if err != nil {
				ServerError(w, err)
				return
			}

			// create the user information and insert it into the db
			user, err := repo.InsertUser(repo.NewUser(emailStr, string(hashedPassword)))
			if err != nil {
				// we do not want duplicate email registrations
				if errors.Is(err, sql.ErrNoRows) {
					ServerError(w, err)
					return
				}
				if liteErr, ok := err.(*sqlite.Error); ok {
					code := liteErr.Code()
					if code == 2067 {
						data.ErrorMessage = "Invalid email or password."
						tmpl.ExecuteTemplate(w, "root", data)
					}
				}
			}

			// save user id into the cookie
			session.Values["userId"] = user.Id
			err = sessions.Save(r, w)
			if err != nil {
				ServerError(w, err)
				return
			}

			// redirect user to today
			http.Redirect(w, r, "/today", http.StatusSeeOther)
		}
	}
}

func (s *Server) handleHistory(view []HTMLFile) http.HandlerFunc {
	type ViewData struct {
		SiteData SiteData
		Meals    []repo.Meal
	}

	tmpl := s.CompileTemplates("index.html", view, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		// get the user id from the cookie
		userId, err := GetUserId(r, s.Sessions)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusUnauthorized)
			return
		}

		var meals []repo.Meal
		// get the date query parameter
		dateStr := r.URL.Query().Get("date")

		if dateStr != "" {
			// parse the date
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				http.Error(w, "Invalid date format", http.StatusBadRequest)
				return
			}
			meals, err = repo.GetMealsByUserAndDate(repo.User{Id: userId}, date)
			if err != nil {
				ServerError(w, err)
				return
			}
		} else {
			var err error
			meals, err = repo.GetAllMeals()
			if err != nil {
				ServerError(w, err)
				return
			}
		}

		tmpl.ExecuteTemplate(w, "root", ViewData{
			SiteData: s.SiteData,
			Meals:    meals,
		})
	}
}

// this is called by HTMX
func (s *Server) handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			// delete the session by removing the user id from session values
			session, _ := s.Sessions.Get(r, "session")
			delete(session.Values, "userId")
			err := sessions.Save(r, w)
			if err != nil {
				ServerError(w, err)
				return
			}
			// send the user back to login
			w.Header().Add("HX-Redirect", "/login")
		}
	}
}

func (s *Server) handleMeals() http.HandlerFunc {
	type FormData struct {
		Name     string
		MealType repo.MealType
	}

	return func(w http.ResponseWriter, r *http.Request) {
		userId, err := GetUserId(r, s.Sessions)
		if err != nil {
			ServerError(w, err)
			return
		}

		err = r.ParseForm()
		if err != nil {
			ServerError(w, err)
			return
		}

		meals := []repo.MealType{
			repo.Breakfast,
			repo.Lunch,
			repo.Dinner,
			repo.Snacks,
		}

		var data FormData
		for _, meal := range meals {
			str := r.Form.Get(string(meal))
			if str != "" {
				data.Name = str
				data.MealType = meal
				break
			}
		}

		if data.Name == "" {
			http.Error(w, "Error, recieved an empty form submission!", http.StatusBadRequest)
			return
		}

		// create and insert meal record into the database
		_, err = repo.InsertMeal(repo.NewMeal(data.Name, userId, data.MealType, time.Now()))
		if err != nil {
			ServerError(w, err)
			return
		}

		// re-render the today page by redirect
		http.Redirect(w, r, "/today", http.StatusSeeOther)
	}
}

func (s *Server) handleDeleteMeal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		userId, err := GetUserId(r, s.Sessions)
		if err != nil {
			ServerError(w, err)
			return
		}

		err = repo.DeleteMealByUserAndId(repo.User{Id: userId}, id)
		if err != nil {
			ServerError(w, err)
			return
		}

		w.Header().Add("HX-Redirect", "/today")
	}
}

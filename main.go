package main

import (
	"database/sql"
	"embed"
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

const (
	Port            = "8081"
	StaticDirName   = "static"
	TemplateDirName = "templates"

	// HTML Base Templates
	RootHTML   HTMLFile = "templates/root.html"
	HeadHTML   HTMLFile = "templates/head.html"
	LayoutHTML HTMLFile = "templates/layout.html"

	// HTML Views
	TodayHTML    HTMLFile = "templates/views/today.html"
	HistoryHTML  HTMLFile = "templates/views/history.html"
	LoginHTML    HTMLFile = "templates/views/login.html"
	RegisterHTML HTMLFile = "templates/views/register.html"

	// HTML Components
	NavHTML            HTMLFile = "templates/components/nav.html"
	TableHTMLComponent HTMLFile = "templates/components/table.html"
	ModalHTMLComponent HTMLFile = "templates/components/modal.html"
)

type HTMLFile string

type SiteData struct {
	Title string
}

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

//go:embed templates/* static/*
var embedFS embed.FS

func main() {
	router := chi.NewMux()
	store := sessions.NewCookieStore([]byte("3lWcaN9nYFjh9Dy5RJWXR84nxYSOZSQx4R11y8NxUNQ="))
	siteData := SiteData{Title: "Food Diary"}

	s := Server{
		FileSystem:   embedFS,
		Router:       router,
		Sessions:     store,
		Port:         Port,
		StaticDir:    StaticDirName,
		TemplatesDir: TemplateDirName,
		SiteData:     siteData,
	}

	s.Routes()

	err := repo.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("[ 💿 Spinning up server on http://localhost:" + s.Port + " ]")

	if err = http.ListenAndServe(":"+s.Port, s.Router); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) Routes() {
	s.Router.Handle("/static/*", http.FileServer(http.FS(s.FileSystem)))
	s.Router.HandleFunc("/", s.handleIndex())
	s.Router.HandleFunc("/today", s.handleToday())
	s.Router.HandleFunc("/history", s.handleHistory())
	s.Router.HandleFunc("/login", s.handleLogin())
	s.Router.HandleFunc("/logout", s.handleLogout())
	s.Router.HandleFunc("/register", s.handleRegister())

	s.Router.Post("/api/meals", s.handleMeals())
	s.Router.Delete("/api/meals/{id}", s.handleDeleteMeal())
}

func (s *Server) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/today", http.StatusSeeOther)
	}
}

func (s *Server) handleRegister() http.HandlerFunc {
	type ViewData struct {
		SiteData     SiteData
		ErrorMessage string
	}
	var register = []HTMLFile{
		RootHTML,
		LayoutHTML,
		HeadHTML,
		NavHTML,
		RegisterHTML,
	}

	tmpl := s.CompileTemplates("register.html", register, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		data := ViewData{SiteData: s.SiteData}
		data.SiteData.Title = data.SiteData.Title + " | Register"
		session, _ := s.Sessions.Get(r, "session")
		if r.Method == "POST" {
			r.ParseForm()
			emailStr := r.Form.Get("email")
			passwordStr := r.Form.Get("password")

			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordStr), 10)
			if err != nil {
				log.Print(err)
				http.Error(w, "Something went wrong on our side", http.StatusInternalServerError)
				return
			}

			user := repo.NewUser(emailStr, string(hashedPassword))
			user, err = repo.InsertUser(user)
			if err != nil {
				log.Println(err)
				if errors.Is(err, sql.ErrNoRows) {
					http.Error(w, "Something went wrong on our side", http.StatusInternalServerError)
					return
				}
				if liteErr, ok := err.(*sqlite.Error); ok {
					code := liteErr.Code()
					if code == 2067 {
						data.ErrorMessage = "Error! Email already exists"
						tmpl.ExecuteTemplate(w, "root", data)
					}
				}
			}

			session.Values["userId"] = user.Id
			err = sessions.Save(r, w)
			if err != nil {
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/today", http.StatusSeeOther)
		} else {
			tmpl.ExecuteTemplate(w, "root", data)
		}
	}
}

func (s *Server) handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			session, _ := s.Sessions.Get(r, "session")
			delete(session.Values, "userId")
			err := sessions.Save(r, w)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			w.Header().Add("HX-Redirect", "/login")
		}
	}
}

func (s *Server) handleLogin() http.HandlerFunc {
	type ViewData struct {
		SiteData     SiteData
		ErrorMessage string
	}
	var login = []HTMLFile{
		RootHTML,
		LayoutHTML,
		HeadHTML,
		NavHTML,
		LoginHTML,
	}

	tmpl := s.CompileTemplates("login.html", login, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		data := ViewData{
			SiteData:     s.SiteData,
			ErrorMessage: "",
		}
		data.SiteData.Title = data.SiteData.Title + " | Login"
		session, _ := s.Sessions.Get(r, "session")
		if r.Method == "GET" {
			tmpl.ExecuteTemplate(w, "root", data)
		}
		if r.Method == "POST" {
			r.ParseForm()
			emailStr := r.Form.Get("email")
			passwordStr := r.Form.Get("password")

			user, err := repo.GetUserByEmail(emailStr)
			if err != nil {
				log.Print(err)
				data.ErrorMessage = "Invalid email or password"
				tmpl.ExecuteTemplate(w, "root", data)
				return
			}

			err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(passwordStr))
			if err != nil {
				if err == sql.ErrNoRows {
					log.Print(err)
				} else {
					log.Print(err)
				}
				data.ErrorMessage = "Invalid email or password"
				tmpl.ExecuteTemplate(w, "root", data)
				return
			}

			session.Values["userId"] = user.Id
			err = sessions.Save(r, w)
			if err != nil {
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			log.Println("login success")
			http.Redirect(w, r, "/today", http.StatusSeeOther)
		}
	}
}

func (s *Server) handleToday() http.HandlerFunc {
	type ViewData struct {
		SiteData SiteData
		Meals    []repo.Meal
	}

	var today = []HTMLFile{
		HeadHTML,
		LayoutHTML,
		RootHTML,
		NavHTML,
		TodayHTML,
		TableHTMLComponent,
		ModalHTMLComponent,
	}

	var data ViewData
	data.SiteData = s.SiteData

	tmpl := s.CompileTemplates("today.html", today, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.Sessions.Get(r, "session")
		var user repo.User
		switch v := session.Values["userId"].(type) {
		case int64:
			user.Id = v
		case nil:
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		default:
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		meals, err := repo.GetMealsByUserAndDate(user, time.Now())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.Meals = meals
		tmpl.ExecuteTemplate(w, "root", data)
	}
}

func (s *Server) handleMeals() http.HandlerFunc {
	type FormData struct {
		Name     string
		MealType repo.MealType
	}

	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.Sessions.Get(r, "session")

		var user repo.User
		switch v := session.Values["userId"].(type) {
		case int64:
			user.Id = v
		case nil:
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		default:
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		err := r.ParseForm()
		if err != nil {
			log.Print(err)
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

		newMeal := repo.NewMeal(data.Name, user.Id, data.MealType, time.Now())
		log.Println(newMeal)

		newMeal, err = repo.InsertMeal(newMeal)
		if err != nil {
			http.Error(w, "Error inserting meal.", http.StatusInternalServerError)
			return
		}
		log.Println("added", newMeal)

		http.Redirect(w, r, "/today", http.StatusSeeOther)
	}

}

func (s *Server) handleHistory() http.HandlerFunc {
	type ViewData struct {
		SiteData SiteData
		Meals    []repo.Meal
	}

	var index = []HTMLFile{
		HeadHTML,
		LayoutHTML,
		RootHTML,
		NavHTML,
		HistoryHTML,
		TableHTMLComponent,
	}

	var data ViewData
	data.SiteData = s.SiteData

	tmpl := s.CompileTemplates("index.html", index, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := s.Sessions.Get(r, "session")

		var user repo.User
		switch v := session.Values["userId"].(type) {
		case int64:
			user.Id = v
		case nil:
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		default:
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		var meals []repo.Meal
		dateStr := r.URL.Query().Get("date")
		if dateStr != "" {
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				http.Error(w, "Invalid date format", http.StatusBadRequest)
				return
			}
			meals, err = repo.GetMealsByUserAndDate(user, date)
			if err != nil {
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			var err error
			meals, err = repo.GetAllMeals()
			if err != nil {
				log.Print(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		data.Meals = meals

		tmpl.ExecuteTemplate(w, "root", data)
	}
}

func (s *Server) handleDeleteMeal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")

		session, _ := s.Sessions.Get(r, "session")
		var user repo.User
		switch v := session.Values["userId"].(type) {
		case int64:
			user.Id = v
		case nil:
			http.Error(w, "Invalid user id", http.StatusUnauthorized)
			return
		default:
			http.Error(w, "Invalid user id", http.StatusUnauthorized)
			return
		}

		err := repo.DeleteMealByUserAndId(user, id)
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("HX-Redirect", "/today")
	}
}

func (s *Server) CompileTemplates(name string, files []HTMLFile, funcMap template.FuncMap) *template.Template {
	tmpl := template.New(name)

	if funcMap != nil {
		tmpl.Funcs(funcMap)
	}

	var patterns []string
	for _, file := range files {
		patterns = append(patterns, string(file))
	}

	tmpl, err := tmpl.ParseFS(s.FileSystem, patterns...)
	if err != nil {
		log.Fatal(err)
	}

	return tmpl
}

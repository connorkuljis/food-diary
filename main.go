package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"text/template"
	"time"

	"github.com/connorkuljis/food-diary/repo"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
)

const SessionName = "session"

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

//go:embed templates/* static/*
var inMemoryFS embed.FS

type HTMLFile string

const (
	RootHTML   HTMLFile = "templates/root.html"
	HeadHTML   HTMLFile = "templates/head.html"
	LayoutHTML HTMLFile = "templates/layout.html"
)

func main() {
	fileSystem := inMemoryFS
	router := chi.NewMux()
	store := sessions.NewCookieStore([]byte("special_key"))
	port := "8081"
	staticDir := "static"
	templateDir := "templates"
	siteData := SiteData{
		Title: "Food Diary",
	}

	log.Println("[ 💿 Spinning up server on http://localhost:" + port + " ]")

	err := repo.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	s := Server{
		FileSystem:   fileSystem,
		Router:       router,
		Sessions:     store,
		Port:         port,
		StaticDir:    staticDir,
		TemplatesDir: templateDir,
		SiteData:     siteData,
	}

	s.Routes()

	err = http.ListenAndServe(":"+s.Port, s.Router)
	if err != nil {
		panic(err)
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

func (s *Server) Routes() {
	s.Router.Handle("/static/*", http.FileServer(http.FS(s.FileSystem)))
	s.Router.HandleFunc("/", s.handleIndex())
	s.Router.Post("/meals", s.handleMeals())
	s.Router.Delete("/meals/{id}", s.handleDeleteMeal())
	// s.Router.HandleFunc("/meals/{date}", s.handleMealsByDate())
}

func (s *Server) handleIndex() http.HandlerFunc {
	type ViewData struct {
		SiteData SiteData
		Meals    []repo.Meal
	}

	var index = []HTMLFile{
		HeadHTML,
		LayoutHTML,
		RootHTML,
	}

	var data ViewData
	data.SiteData = s.SiteData

	tmpl := s.CompileTemplates("index.html", index, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		meals, err := repo.GetAllMeals()
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

		newMeal := repo.NewMeal(data.Name, data.MealType, time.Now())
		log.Println(newMeal)

		newMeal, err = repo.InsertMeal(newMeal)
		if err != nil {
			http.Error(w, "Error inserting meal.", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	}

}

func (s *Server) handleMealsByDate() http.HandlerFunc {
	type ViewData struct {
		SiteData SiteData
		Meals    []repo.Meal
	}

	var index = []HTMLFile{
		HeadHTML,
		LayoutHTML,
		RootHTML,
	}

	var data ViewData
	data.SiteData = s.SiteData

	tmpl := s.CompileTemplates("index.html", index, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		dateStr := chi.URLParam(r, "date")
		log.Print("date", dateStr)
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			http.Error(w, "Invalid date format", http.StatusBadRequest)
			return
		}

		meals, err := repo.GetMealsByDate(date)
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		log.Println(meals)

		data.Meals = meals

		tmpl.ExecuteTemplate(w, "root", data)
	}
}

func (s *Server) handleDeleteMeal() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		log.Println("entered delete")

		err := repo.DeleteMealByID(id)
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("HX-Redirect", "/")
	}
}

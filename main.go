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

const (
	RootHTML   HTMLFile = "templates/root.html"
	HeadHTML   HTMLFile = "templates/head.html"
	LayoutHTML HTMLFile = "templates/layout.html"

	NavHTML HTMLFile = "templates/components/nav.html"

	TodayHTML   HTMLFile = "templates/views/today.html"
	HistoryHTML HTMLFile = "templates/views/history.html"

	TableHTMLComponent HTMLFile = "templates/components/table.html"
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

//go:embed templates/* static/*
var inMemoryFS embed.FS

type HTMLFile string

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
	s.Router.HandleFunc("/today", s.handleToday())
	s.Router.HandleFunc("/history", s.handleHistory())
	s.Router.Post("/api/meals", s.handleMeals())
	s.Router.Delete("/api/meals/{id}", s.handleDeleteMeal())
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
	}

	var data ViewData
	data.SiteData = s.SiteData

	tmpl := s.CompileTemplates("today.html", today, nil)

	return func(w http.ResponseWriter, r *http.Request) {
		meals, err := repo.GetMealsByDate(time.Now())
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

		if data.Name == "" {
			http.Error(w, "Error, recieved an empty form submission!", http.StatusBadRequest)
			return
		}

		newMeal := repo.NewMeal(data.Name, data.MealType, time.Now())
		log.Println(newMeal)

		newMeal, err = repo.InsertMeal(newMeal)
		if err != nil {
			http.Error(w, "Error inserting meal.", http.StatusInternalServerError)
			return
		}

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
		var meals []repo.Meal
		dateStr := r.URL.Query().Get("date")
		if dateStr != "" {
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				http.Error(w, "Invalid date format", http.StatusBadRequest)
				return
			}
			meals, err = repo.GetMealsByDate(date)
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
		log.Println("entered delete")

		err := repo.DeleteMealByID(id)
		if err != nil {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("HX-Redirect", "/today")
	}
}

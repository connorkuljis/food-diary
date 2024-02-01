package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/connorkuljis/food-diary/repo"
	"github.com/connorkuljis/food-diary/server"
)

//go:embed templates/* static/*
var embedFS embed.FS

func main() {
	s := server.NewServer(embedFS)

	s.Routes()

	if err := repo.InitDB(); err != nil {
		log.Fatal(err)
	}

	log.Println("[ 💿 Spinning up server on http://localhost:" + s.Port + " ]")

	if err := http.ListenAndServe(":"+s.Port, s.Router); err != nil {
		log.Fatal(err)
	}
}

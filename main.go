package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
)

//go:embed views/*.html
var viewsFS embed.FS

//go:embed css/* assets/*
var staticFS embed.FS

func main() {
	port := flag.Int("port", 7272, "Port for HTTP server")
	flag.Parse()
	logger := defaultLogger()
	templates := newTemplates()
	mux := http.NewServeMux()

	db, dbErr := NewSqliteClient("ppacer_ff.db", logger)
	if dbErr != nil {
		logger.Error("Cannot create database client", "err", dbErr.Error())
		panic(dbErr)
	}
	owner := NewOwner(db, logger, templates)

	mux.Handle("/css/", http.FileServer(http.FS(staticFS)))
	mux.Handle("/assets/", http.FileServer(http.FS(staticFS)))
	mux.HandleFunc("/", owner.MainHandler)
	mux.HandleFunc("GET /health", owner.HealthHandler)
	mux.HandleFunc("POST /register", owner.RegistrationHandler)
	mux.HandleFunc("GET /confirm/{hash}", owner.ConfirmHandler)
	mux.HandleFunc("/policy", owner.PolicyHandler)

	portStr := fmt.Sprintf(":%d", *port)
	fmt.Println("Listening on port", portStr)
	lErr := http.ListenAndServe(portStr, mux)
	if lErr != nil {
		logger.Error("Cannot start new server", "err", "lErr")
		panic(lErr)
	}
}

type templates struct {
	templates *template.Template
}

func (t *templates) Render(w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplates() *templates {
	return &templates{
		templates: template.Must(template.ParseFS(viewsFS, "views/*.html")),
	}
}

package main

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"time"
)

//go:embed views/*.html
var viewsFS embed.FS

//go:embed css/*
var staticFS embed.FS

type User struct {
	Email          string
	Nickname       string
	Hash           string
	RegistrationTs time.Time
	Confirmed      bool
	ConfirmationTs time.Time
}

type page struct {
	ShowForm          bool
	PostRegisterInfo  string
	PostRegisterError string
}

func main() {
	logger := slog.Default()
	mux := http.NewServeMux()
	templates := newTemplates()
	secrets, awsErr := getSecretFromAWS()
	if awsErr != nil {
		panic(awsErr)
	}

	// db maps email to User info.
	db := make(map[string]User)

	mux.Handle("/css/", http.FileServer(http.FS(staticFS)))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderErr := templates.Render(w, "index", page{ShowForm: true})
		if renderErr != nil {
			logger.Error("Cannot render <index>", "err", renderErr.Error())
		}
	})

	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		email := r.FormValue("email")
		nickname := r.FormValue("nickname")

		if user, exists := db[email]; exists {
			var errMsg string
			if user.Confirmed {
				errMsg = fmt.Sprintf("Person using email [%s] is already registered, thank you!",
					email)
			} else {
				errMsg = fmt.Sprintf("Person using email [%s] is already registered "+
					"but didn't confirm their email. Please check your inbox and spam folder.",
					email)
			}
			p := page{
				PostRegisterError: errMsg,
			}
			renderErr := templates.Render(w, "notifications", p)
			if renderErr != nil {
				logger.Error("Cannot render <index>", "err", renderErr.Error())
			}
			return
		}

		now := time.Now()
		hash := userHash(email, now)
		user := User{
			Email:          email,
			Nickname:       nickname,
			Hash:           hash,
			RegistrationTs: now,
			Confirmed:      false,
		}
		db[email] = user
		sendEmail(
			email,
			"ppacer preview: friends&family - email confirmation",
			fmt.Sprintf("Please confirm your email by clicking the link: https://ff.ppacer.org/confirm/%s",
				hash),
			secrets,
		)

		msg := fmt.Sprintf("Thank you for registering! Please check your inbox and confirm your email (%s).",
			email)
		p := page{PostRegisterInfo: msg}

		renderErr := templates.Render(w, "notifications", p)
		if renderErr != nil {
			logger.Error("Cannot render <index>", "err", renderErr.Error())
		}
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		for email, user := range db {
			fmt.Fprintf(w, "%s: %+v\n", email, user)
		}
	})

	mux.HandleFunc("GET /confirm/{hash}", func(w http.ResponseWriter, r *http.Request) {
		confirmHash := r.PathValue("hash")
		confirmed := false
		var email string
		if confirmHash == "" {
			logger.Error("/confirm/{hash}: Expected hash, but got empty value")
			return
		}
		for _, user := range db {
			if user.Hash != confirmHash {
				continue
			}
			db[user.Email] = User{
				Email:          user.Email,
				Nickname:       user.Nickname,
				Hash:           user.Hash,
				RegistrationTs: user.RegistrationTs,
				Confirmed:      true,
				ConfirmationTs: time.Now(),
			}
			logger.Info("User confirmed", "email", user.Email, "hash",
				user.Hash)
			confirmed = true
			email = user.Email
			break
		}
		logger.Info("Hash not found", "hash", confirmHash)

		var p page
		if confirmed {
			p = page{
				ShowForm: false,
				PostRegisterInfo: fmt.Sprintf("Email [%s] has been confirmed. Thank you for registration!",
					email),
			}
		} else {
			p = page{
				PostRegisterError: fmt.Sprintf("Something went wrong. Cannot find hash [%s]. Please contact info@dskrzypiec.dev",
					confirmHash),
			}
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		renderErr := templates.Render(w, "index", p)
		if renderErr != nil {
			logger.Error("Cannot render <index>", "err", renderErr.Error())
		}
	})

	const port = ":7272"
	fmt.Println("Listening on port", port)
	lErr := http.ListenAndServe(port, mux)
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

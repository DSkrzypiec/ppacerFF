package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type User struct {
	Email          string
	Nickname       *string
	Hash           string
	RegistrationTs time.Time
	Confirmed      bool
	ConfirmationTs time.Time
	Drinks         bool
}

type page struct {
	ShowForm          bool
	PostRegisterInfo  string
	PostRegisterError string
}

type Owner struct {
	db       *SqliteDB
	logger   *slog.Logger
	tmpl     *templates
	secrets  emailSecret
	telegram *Telegram
}

func NewOwner(db *SqliteDB, logger *slog.Logger, tmpl *templates) *Owner {
	emailSecret, err := getEmailSecrets()
	if err != nil {
		logger.Error("Cannot get email credentials from AWS", "err",
			err.Error())
		panic(err)
	}
	return &Owner{
		db:       db,
		logger:   logger,
		tmpl:     tmpl,
		secrets:  emailSecret,
		telegram: NewTelegram(),
	}
}

func (o *Owner) MainHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderErr := o.tmpl.Render(w, "index", page{ShowForm: true})
	if renderErr != nil {
		o.logger.Error("Cannot render <index>", "err", renderErr.Error())
	}
}

func (o *Owner) RegistrationHandler(w http.ResponseWriter, r *http.Request) {
	email := r.FormValue("email")
	nickname := r.FormValue("nickname")
	drinks := r.FormValue("drinks")
	drinksBool := drinks == "on"

	userDb, uErr := UserByEmail(o.db, email)
	exists := uErr != ErrUserNotFound
	if uErr != nil && uErr != ErrUserNotFound {
		o.logger.Error("Unexpected error while reading user info", "email",
			email, "err", uErr.Error())
	}
	if exists {
		var errMsg string
		if userDb.Confirmed == 1 {
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
		renderErr := o.tmpl.Render(w, "notifications", p)
		if renderErr != nil {
			o.logger.Error("Cannot render <index>", "err", renderErr.Error())
		}
		return
	}

	now := time.Now()
	hash := userHash(email, now)
	user := User{
		Email:          email,
		Nickname:       &nickname,
		Hash:           hash,
		RegistrationTs: now,
		Drinks:         drinksBool,
		Confirmed:      false,
	}
	iErr := InsertNewUser(o.db, user)
	if iErr != nil {
		o.logger.Error("Cannot insert new user", "user", user, "err",
			iErr.Error())
		o.telegram.Send(
			fmt.Sprintf("[ppacerFF] Cannot insert new user [%s]: %s",
				user.Email, iErr.Error()),
		)
	}
	sendEmail(
		email,
		"ppacer preview: friends&family - email confirmation",
		fmt.Sprintf("Please confirm your email by clicking the link: https://ff.ppacer.org/confirm/%s",
			hash),
		o.secrets,
	)

	msg := fmt.Sprintf("Thank you for registering! Please check your inbox and confirm your email (%s).",
		email)
	p := page{PostRegisterInfo: msg}
	o.telegram.Send(
		fmt.Sprintf("[ppacerFF] New user registered: [%s] - %s", user.Email,
			*user.Nickname),
	)

	renderErr := o.tmpl.Render(w, "notifications", p)
	if renderErr != nil {
		o.logger.Error("Cannot render <index>", "err", renderErr.Error())
	}
}

func (o *Owner) ConfirmHandler(w http.ResponseWriter, r *http.Request) {
	confirmHash := r.PathValue("hash")
	confirmed := false
	var email string
	if confirmHash == "" {
		o.logger.Error("/confirm/{hash}: Expected hash, but got empty value")
		return
	}
	userDb, uErr := UserByHash(o.db, confirmHash)
	if uErr != nil && uErr != ErrUserNotFound {
		o.logger.Error("Unexpected error when reading user by hash",
			"hash", confirmHash, "err", uErr.Error())
	}
	if uErr == nil {
		confirmed = true
		email = userDb.Email
		o.logger.Info("User confirmed", "email", userDb.Email, "hash",
			userDb.Hash)
		o.telegram.Send(
			fmt.Sprintf("[ppacerFF] User [%s] confirmed their email",
				userDb.Email),
		)
		iErr := ConfirmUser(o.db, userDb.Email, userDb.Hash)
		if iErr != nil {
			o.logger.Error("Error while confirming user", "email",
				userDb.Email, "hash", userDb.Hash, "err", iErr.Error())
			o.telegram.Send(
				fmt.Sprintf("[ppacerFF] Error while confirim user [%s]: %s",
					userDb.Email, iErr.Error()),
			)
		}
	}

	var p page
	if confirmed {
		p = page{
			ShowForm: false,
			PostRegisterInfo: fmt.Sprintf("Email [%s] has been confirmed. Thank you for registration!",
				email),
		}
	} else {
		o.logger.Info("Hash not found", "email", email, "hash", confirmHash)
		p = page{
			PostRegisterError: fmt.Sprintf("Something went wrong. Cannot find hash [%s]. Please contact info@dskrzypiec.dev",
				confirmHash),
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderErr := o.tmpl.Render(w, "index", p)
	if renderErr != nil {
		o.logger.Error("Cannot render <index>", "err", renderErr.Error())
	}
}

func (o *Owner) PolicyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	renderErr := o.tmpl.Render(w, "policy", page{})
	if renderErr != nil {
		o.logger.Error("Cannot render <policy>", "err", renderErr.Error())
	}
}

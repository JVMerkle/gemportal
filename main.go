// SPDX-FileCopyrightText: 2021 Julian Merkle <me@jvmerkle.de>
//
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"embed"
	"net/http"

	log "github.com/sirupsen/logrus"

	_ "embed"

	"github.com/JVMerkle/gemportal/app"
	"github.com/gorilla/mux"
)

const AppVersion = "1.0.0-dev"

//go:embed static/style.css
//go:embed static/app.js
var staticFS embed.FS

//go:embed templates/index.html
var templateFS embed.FS

func main() {
	cfg, err := app.GetConfig(AppVersion)
	if err != nil {
		log.Fatalf("Could not load config: %s", err.Error())
	}

	log.SetLevel(log.Level(cfg.LogLevel))

	gemPortal := app.NewGemPortal(*cfg, templateFS)

	catchAllHandleFunc := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Redirect(w, r, cfg.BaseHREF, http.StatusSeeOther)
		} else {
			http.Redirect(w, r, cfg.BaseHREF, http.StatusPermanentRedirect)
		}
	}

	r := mux.NewRouter()
	r.Use(PanicMiddleware)

	// Static files
	r.PathPrefix(cfg.BaseHREF + "static/").Handler(http.StripPrefix(cfg.BaseHREF,
		http.FileServer(http.FS(staticFS))))

	// Application handler
	r.PathPrefix(cfg.BaseHREF).Handler(gemPortal).Methods("GET")

	// Catch-all (e.g. empty path)
	r.PathPrefix("/").HandlerFunc(catchAllHandleFunc)

	listen := "0.0.0.0:" + cfg.HTTPPort
	log.Infof("Listening on '%s'", listen+cfg.BaseHREF)
	log.Fatal(http.ListenAndServe(listen, r))
}

// PanicMiddleware catches panics, logs them and responds with an HTTP 500.
func PanicMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			err := recover()
			if err != nil {
				log.Errorf("Panic Middleware: %s", err)
				http.Error(w, "Gemportal here. There was an internal server error. If the error persists please contact the administrator.", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

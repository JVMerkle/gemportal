package main

import (
	"embed"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"

	_ "embed"

	"github.com/gorilla/mux"
)

const AppVersion = "0.0.3"

//go:embed static/style.css
//go:embed static/app.js
var staticFS embed.FS

//go:embed templates/index.html
var templateFS embed.FS

type CatchAllHandler struct {
	cfg *Cfg
}

func NewCatchAllHandler(cfg *Cfg) *CatchAllHandler {
	return &CatchAllHandler{
		cfg: cfg,
	}
}

func (cah *CatchAllHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Redirect(w, r, cah.cfg.BaseHREF, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, cah.cfg.BaseHREF, http.StatusPermanentRedirect)
	}
}

func main() {
	cfg, err := GetConfig(AppVersion)
	if err != nil {
		panic(fmt.Sprintf("error loading config: %s", err.Error()))
	}

	log.SetLevel(log.Level(cfg.LogLevel))

	gemPortal := NewGemPortal(cfg)
	catchAllHandler := NewCatchAllHandler(cfg)

	r := mux.NewRouter()

	// Static files
	r.PathPrefix(cfg.BaseHREF + "static/").Handler(http.StripPrefix(cfg.BaseHREF,
		http.FileServer(http.FS(staticFS))))

	// Application handler
	r.PathPrefix(cfg.BaseHREF).Handler(gemPortal).Methods("GET")

	// Catch-all (e.g. empty path)
	r.PathPrefix("/").Handler(catchAllHandler)

	listen := ":" + cfg.HTTPPort
	log.Infof("Listening on '%s' with base HREF '%s'", listen, cfg.BaseHREF)
	log.Fatal(http.ListenAndServe(listen, r))
}

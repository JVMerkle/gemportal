package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"

	"git.sr.ht/~yotam/go-gemini"
	"github.com/gorilla/mux"
)

var cfg *Cfg
var indexTemplate *template.Template
var styleCSS []byte
var scriptJS []byte

func errResp(ctx *GemContext, errorText string, httpStatusCode int) {
	ctx.w.WriteHeader(httpStatusCode)

	if len(errorText) > 2 {
		ctx.GemError = strings.ToUpper(errorText[0:1]) + errorText[1:]
	} else {
		ctx.GemError = errorText
	}

	if httpStatusCode >= 500 {
		log.Warnf("Replying with '%s' on requesting '%s': %s", http.StatusText(httpStatusCode), ctx.GemURL.String(), errorText)
	}
	indexTemplate.Execute(ctx.w, ctx)
}

type CatchAllHandler struct {
}

func (cah *CatchAllHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Redirect(w, r, cfg.BaseHREF, http.StatusSeeOther)
	} else {
		http.Redirect(w, r, cfg.BaseHREF, http.StatusPermanentRedirect)
	}
}

type GemPortalHandler struct {
}

// Base handler
func (gph *GemPortalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewGemContext(cfg, w, r)

	// Application root
	if r.URL.Path == ctx.BaseHREF {
		indexTemplate.Execute(w, ctx)
		return
	}

	// Remove the base HREF from the requested path
	ctx.r.URL.Path = r.URL.Path[len(ctx.BaseHREF):]
	path := ctx.r.URL.Path

	if path == "favicon.ico" {
		http.NotFound(w, r)
	} else {
		gph.ServeGemini2HTML(ctx)
	}
}

// Handles Gemini2HTML requests
func (gph *GemPortalHandler) ServeGemini2HTML(ctx *GemContext) {

	geminiURL := ctx.r.URL.Path

	parsedURL, err := parseGeminiURL(ctx, geminiURL)
	if err != nil {
		errResp(ctx, err.Error(), http.StatusBadRequest)
		return
	}
	ctx.GemURL = *parsedURL

	if values, ok := ctx.r.URL.Query()["unsafe"]; ok && len(values) > 0 && values[0] == "on" {
		ctx.DisableTLSChecks = true
	}

	client := gemini.DefaultClient
	if ctx.DisableTLSChecks {
		client.InsecureSkipVerify = true
	}
	res, err := client.Fetch(ctx.GemURL.String())
	if err != nil {
		errResp(ctx, "Could not perform gemini request: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()

	if res.Status != gemini.StatusSuccess {
		errResp(ctx, fmt.Sprintf("Gemini upstream reported status %d", res.Status), http.StatusBadGateway)
		return
	}

	html, err := gemResponseToHTML(ctx, &res)
	if err != nil {
		errResp(ctx, "Error processing Gemini response", http.StatusInternalServerError)
		return
	}

	ctx.GemContent = html
	indexTemplate.Execute(ctx.w, ctx)
}

func main() {
	var err error

	cfg, err = GetConfig()
	if err != nil {
		panic(fmt.Sprintf("error loading config: %s", err.Error()))
	}

	log.SetLevel(log.Level(cfg.LogLevel))

	indexTemplate, err = template.ParseFiles("templates/index.html")
	if err != nil {
		panic(fmt.Sprintf("error parsing index template: %s", err.Error()))
	}

	styleCSS = mustReadToMemory("static/style.css")
	scriptJS = mustReadToMemory("static/app.js")

	r := mux.NewRouter()

	// Static files
	r.PathPrefix(cfg.BaseHREF + "static/").Handler(http.StripPrefix(cfg.BaseHREF+"static/",
		http.FileServer(http.Dir("static/"))))

	// Application handler
	r.PathPrefix(cfg.BaseHREF).Handler(&GemPortalHandler{}).Methods("GET")

	// Catch-all (e.g. empty path)
	r.PathPrefix("/").Handler(&CatchAllHandler{})

	listen := ":" + cfg.HTTPPort
	log.Infof("Listening on '%s' with base HREF '%s'", listen, cfg.BaseHREF)
	log.Fatal(http.ListenAndServe(listen, r))
}

func mustReadToMemory(name string) []byte {
	buf := &bytes.Buffer{}
	if f, err := os.Open(name); err != nil {
		panic(err)
	} else {
		if _, err := io.Copy(buf, f); err != nil {
			panic(fmt.Sprintf("could not read '%s' to memory: %s", name, err.Error()))
		}
	}
	return buf.Bytes()
}

package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"

	_ "embed"

	"git.sr.ht/~yotam/go-gemini"
	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
	"github.com/temoto/robotstxt"
)

const AppVersion = "0.0.1"

//go:embed static/style.css
//go:embed static/app.js
var staticFS embed.FS

//go:embed templates/index.html
var templateFS embed.FS

var cfg *Cfg
var indexTemplate *template.Template
var robotsCache *cache.Cache

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

var (
	ErrRobotsTxtNotFound = errors.New("robots.txt not found")
)

// DownloadRobotsTxt downloads the robots.txt of the ctx.GemURL.Host
func (gph *GemPortalHandler) DownloadRobotsTxt(ctx *GemContext) ([]byte, error) {
	robotsURL := ctx.GemURL
	robotsURL.Path = "/robots.txt"

	client := gemini.DefaultClient
	client.InsecureSkipVerify = true

	res, err := client.Fetch(robotsURL.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.Status == gemini.StatusNotFound {
		return nil, ErrRobotsTxtNotFound
	} else if res.Status != gemini.StatusSuccess {
		return nil, fmt.Errorf("could not retrieve robots.txt (code %d)", res.Status)
	}

	buf := &bytes.Buffer{}
	if n, err := io.CopyN(buf, res.Body, 4096); err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.New("could not retrieve robots.txt due to an io error")
	} else if n == 4096 {
		return nil, errors.New("could not retrieve robots.txt because it exceeds the size limit")
	}

	return buf.Bytes(), nil
}

// IsWebproxyAllowed checks if the webproxy is allowed to request the
// resource (ctx.GemURL) as specified in the hosts robots.txt
func (gph *GemPortalHandler) IsWebproxyAllowed(ctx *GemContext) (bool, error) {
	var robotBytes []byte
	var err error

	// Lookup Host (including Port)
	if val, ok := robotsCache.Get(ctx.GemURL.Host); ok {
		log.Debugf("Robots cache hit for '%s'", ctx.GemURL.Host)
		robotBytes = val.([]byte)
	} else { // Download robots.txt
		log.Debugf("Robots cache miss for '%s'", ctx.GemURL.Host)
		robotBytes, err = gph.DownloadRobotsTxt(ctx)
		if errors.Is(err, ErrRobotsTxtNotFound) {
			robotBytes = []byte{}
		} else if err != nil {
			return false, err
		}
		robotsCache.Set(ctx.GemURL.Host, robotBytes, cache.DefaultExpiration)
	}

	robots, err := robotstxt.FromBytes(robotBytes)
	if err != nil {
		robotsCache.Delete(ctx.GemURL.Host)
		return false, errors.New("unable to parse robots.txt")
	}

	return robots.TestAgent(ctx.GemURL.Path, "webproxy"), nil
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

	if allowed, err := gph.IsWebproxyAllowed(ctx); err != nil {
		errResp(ctx, fmt.Sprintf("Error checking the webproxy permissions: %s", err.Error()), http.StatusBadRequest)
		log.Error(err)
		return
	} else if !allowed {
		errResp(ctx, "Host does not allow webproxies on this path", http.StatusForbidden)
		return
	}

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

	cfg, err = GetConfig(AppVersion)
	if err != nil {
		panic(fmt.Sprintf("error loading config: %s", err.Error()))
	}

	log.SetLevel(log.Level(cfg.LogLevel))

	indexTemplate, err = template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		panic(fmt.Sprintf("error parsing index template: %s", err.Error()))
	}

	// Create a cache with a default expiration time of 24 hours, and which
	// purges expired items every 12 hours
	robotsCache = cache.New(24*time.Hour, 12*time.Hour)

	r := mux.NewRouter()

	// Static files
	r.PathPrefix(cfg.BaseHREF + "static/").Handler(http.StripPrefix(cfg.BaseHREF,
		http.FileServer(http.FS(staticFS))))

	// Application handler
	r.PathPrefix(cfg.BaseHREF).Handler(&GemPortalHandler{}).Methods("GET")

	// Catch-all (e.g. empty path)
	r.PathPrefix("/").Handler(&CatchAllHandler{})

	listen := ":" + cfg.HTTPPort
	log.Infof("Listening on '%s' with base HREF '%s'", listen, cfg.BaseHREF)
	log.Fatal(http.ListenAndServe(listen, r))
}

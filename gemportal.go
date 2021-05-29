package main

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"

	"code.rocketnine.space/tslocum/gmitohtml/pkg/gmitohtml"
	"git.sr.ht/~yotam/go-gemini"
	gem "github.com/JVMerkle/gemportal/gemini"
	"github.com/microcosm-cc/bluemonday"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/temoto/robotstxt"
)

//go:embed templates/index.html
var templateFS embed.FS

var urlRegexp *regexp.Regexp

func init() {
	urlRegex := `=> +([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`
	urlRegexp = regexp.MustCompile(urlRegex)
}

type GemPortal struct {
	cfg           *Cfg
	robotsCache   *cache.Cache
	indexTemplate *template.Template
}

func NewGemPortal(cfg *Cfg) *GemPortal {
	// Create a cache with a default expiration time of 24 hours, and which
	// purges expired items every 12 hours
	robotsCache := cache.New(24*time.Hour, 12*time.Hour)

	indexTemplate, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		panic(fmt.Sprintf("error parsing index template: %s", err.Error()))
	}

	return &GemPortal{
		cfg:           cfg,
		robotsCache:   robotsCache,
		indexTemplate: indexTemplate,
	}
}

// Base handler
func (gp *GemPortal) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := NewReqContext(gp.cfg, w, r)

	// Application root
	if r.URL.Path == ctx.BaseHREF {
		gp.indexTemplate.Execute(w, ctx)
		return
	}

	// Remove the base HREF from the requested path
	ctx.r.URL.Path = r.URL.Path[len(ctx.BaseHREF):]
	path := ctx.r.URL.Path

	if path == "favicon.ico" {
		http.NotFound(w, r)
	} else {

		// Store the requested Gemini URL in the context
		parsedURL, err := parseGeminiURL(ctx, ctx.r.URL.Path)
		if err != nil {
			gp.errResp(ctx, err.Error(), http.StatusBadRequest)
			return
		}
		ctx.GemURL = *parsedURL

		gp.ServeGemini2HTML(ctx)
	}
}

// DownloadRobotsTxt downloads the robots.txt of the ctx.GemURL.Host
func (gp *GemPortal) DownloadRobotsTxt(ctx *ReqContext) ([]byte, error) {
	robotsURL := ctx.GemURL
	robotsURL.Path = "/robots.txt"

	client := gemini.DefaultClient
	client.InsecureSkipVerify = true

	res, err := client.Fetch(robotsURL.String())
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.Status != gemini.StatusSuccess {
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
// resource (ctx.GemURL) as specified in the hosts robots.txt. If in doubt
// (e.g. robots.txt can not be retrieved) IsWebproxyAllowed returns true (thus
// allowing access)
func (gp *GemPortal) IsWebproxyAllowed(ctx *ReqContext) bool {
	var robots *robotstxt.RobotsData

	// Lookup Host (including Port)
	if val, ok := gp.robotsCache.Get(ctx.GemURL.Host); ok {
		log.Debugf("Robots cache hit for '%s'", ctx.GemURL.Host)
		robots = val.(*robotstxt.RobotsData)

	} else { // Download and parse robots.txt

		log.Debugf("Robots cache miss for '%s'", ctx.GemURL.Host)
		robotBytes, err := gp.DownloadRobotsTxt(ctx)
		if err != nil {
			robotBytes = []byte{}
		}

		robots, err = robotstxt.FromBytes(robotBytes)
		if err != nil {
			gp.robotsCache.Delete(ctx.GemURL.Host)
			log.Warnf("Unable to parse robots.txt of '%s'", ctx.GemURL.Host)
		}

		gp.robotsCache.Set(ctx.GemURL.Host, robots, cache.DefaultExpiration)
	}

	return robots.TestAgent(ctx.GemURL.Path, "webproxy")
}

// Handles Gemini2HTML requests
func (gp *GemPortal) ServeGemini2HTML(ctx *ReqContext) {

	if allowed := gp.IsWebproxyAllowed(ctx); !allowed {
		gp.errResp(ctx, "The host does not allow webproxies on this path", http.StatusForbidden)
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
		gp.errResp(ctx, "Could not perform gemini request: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()

	if gemini.SimplifyStatus(res.Status) == gemini.StatusRedirect {
		gp.redirectHandler(ctx, &res)
		return
	} else if res.Status != gemini.StatusSuccess {
		gp.errResp(ctx, fmt.Sprintf("Gemini upstream reported '%s' (%d)", gem.StatusText(res.Status), res.Status), http.StatusBadGateway)
		return
	}

	html, err := gp.gemResponseToHTML(ctx, &res)
	if err != nil {
		gp.errResp(ctx, "Error processing Gemini response", http.StatusInternalServerError)
		return
	}

	ctx.GemContent = html
	gp.indexTemplate.Execute(ctx.w, ctx)
}

// redirectHandler handles Gemini redirects
func (gp *GemPortal) redirectHandler(ctx *ReqContext, res *gemini.Response) {
	if gemini.SimplifyStatus(res.Status) != gemini.StatusRedirect {
		panic("redirectHandler is intended for 'redirect gemini.Response' only")
	}

	var newURL *url.URL

	redirectGemURL, err := gemParseURL(ctx, res.Meta)
	if err == nil {
		newURL, err = url.Parse(redirectGemURL)
	}

	if err != nil {
		gp.errResp(ctx, fmt.Sprintf("Invalid %s to '%s'", gem.StatusText(res.Status), res.Meta), http.StatusBadGateway)
		return
	}

	ctx.GemURL = *newURL

	ctx.redirects += 1
	if ctx.redirects > gp.cfg.GemRedirectsLimit {
		gp.errResp(ctx, "Too many redirects", http.StatusBadGateway)
	} else {
		gp.ServeGemini2HTML(ctx)
	}
}

func (gp *GemPortal) errResp(ctx *ReqContext, errorText string, httpStatusCode int) {
	ctx.w.WriteHeader(httpStatusCode)

	if len(errorText) > 2 {
		ctx.GemError = strings.ToUpper(errorText[0:1]) + errorText[1:]
	} else {
		ctx.GemError = errorText
	}

	if httpStatusCode >= 500 {
		log.Warnf("Replying with '%s' on requesting '%s': %s", http.StatusText(httpStatusCode), ctx.GemURL.String(), errorText)
	}
	gp.indexTemplate.Execute(ctx.w, ctx)
}

// gemResponseToString reads gemtext to a length limited string (30MiB)
func (gp *GemPortal) gemResponseToString(ctx *ReqContext, res *gemini.Response) (string, error) {
	buf := &bytes.Buffer{}

	limit := gp.cfg.GemRespMemLimit

	n, err := io.CopyN(buf, res.Body, limit)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	if n == limit {
		log.Warnf("Limit reached (%d bytes) with a Gemini response", limit)
		return "", ErrGeminiResponseLimit
	}

	return buf.String(), nil
}

// gemResponseToHTML turns Gemtext to safe HTML and rewrites
// all Gemini URLs to hit the application server.
func (gp *GemPortal) gemResponseToHTML(ctx *ReqContext, res *gemini.Response) (string, error) {

	s, err := gp.gemResponseToString(ctx, res)
	if err != nil {
		return "", err
	}

	s = urlRegexp.ReplaceAllStringFunc(s, func(s string) string {
		oldURL := s

		// Strip `=> +`
		s = strings.TrimLeft(s[2:], " ")

		gemURL, err := gemParseURL(ctx, s)
		if err != nil { // Omit URL
			return oldURL
		}

		// Remove the scheme and leading slashes
		gemURL = strings.TrimPrefix(gemURL, "gemini://")
		gemURL = strings.TrimLeft(gemURL, "/")

		// Prepend the base HREF
		newURL := ctx.BaseHREF + gemURL

		log.Debugf("Rewriting URL from '%s' to '%s'", oldURL, newURL)
		return "=> " + newURL
	})

	maybeUnsafeHTML := gmitohtml.Convert([]byte(s), ctx.GemURL.String())
	html := bluemonday.UGCPolicy().SanitizeBytes(maybeUnsafeHTML)

	return string(html), nil
}

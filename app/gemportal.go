// SPDX-FileCopyrightText: 2021-2024 Julian Merkle <me@jvmerkle.de>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"text/template"
	"time"

	"code.rocketnine.space/tslocum/gmitohtml/pkg/gmitohtml"
	"github.com/makeworld-the-better-one/go-gemini"
	"github.com/microcosm-cc/bluemonday"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/temoto/robotstxt"
	"mime"
)

var urlRegexp *regexp.Regexp

func init() {
	urlRegex := `=> +([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`
	urlRegexp = regexp.MustCompile(urlRegex)
}

type GemPortal struct {
	cfg           Config
	robotsCache   *cache.Cache
	indexTemplate *template.Template
}

func NewGemPortal(cfg Config, templateFS fs.FS) *GemPortal {
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
	ctx := NewContext(gp.cfg, w, r)

	// Application root
	if r.URL.Path == ctx.Cfg.BaseHREF {
		gp.executeIndexTemplate(ctx)
		return
	}

	// Filter unknown / handle query values.
	// This is mandatory because later URLs in the Gemini response
	// are rewritten and the current query values are appended to
	// each URL.
	newQuery := make(url.Values)
	for k, v := range r.URL.Query() {
		var reject bool
		switch k {
		case "insecure":
		case "raw":
			ctx.Raw = true
			reject = true
		case "query":
			ctx.GemInputRequest = v[0]
			reject = true
		default:
			reject = true
		}

		if !reject {
			newQuery.Add(k, v[0])
		}
	}
	r.URL.RawQuery = newQuery.Encode()

	// Remove the base HREF from the requested path
	ctx.r.URL.Path = r.URL.Path[len(ctx.Cfg.BaseHREF):]
	path := ctx.r.URL.Path

	if path == "favicon.ico" {
		http.NotFound(w, r)
	} else {

		// Input (if available)
		var input string
		if query, ok := ctx.r.URL.Query()["query"]; ok && len(query) > 0 {
			input = "?query=" + url.QueryEscape(query[0])
		}

		// Store the requested Gemini URL in the context
		parsedURL, err := parseGeminiURL(ctx, ctx.r.URL.Path+input)
		if err != nil {
			gp.errResp(ctx, err.Error(), http.StatusBadRequest)
			return
		}
		ctx.GemURL = *parsedURL

		gp.ServeGemini2HTML(ctx)
	}
}

// DownloadRobotsTxt downloads the robots.txt of the ctx.GemURL.Host
func (gp *GemPortal) DownloadRobotsTxt(ctx *Context) ([]byte, error) {
	robotsURL := ctx.GemURL
	robotsURL.Path = "/robots.txt"

	client := gemini.DefaultClient
	client.Insecure = true

	res, err := client.Fetch(robotsURL.String())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

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

// IsWebProxyAllowed checks if the web proxy is allowed to request the
// resource (ctx.GemURL) as specified in the hosts robots.txt. If in doubt
// (e.g. robots.txt can not be retrieved) IsWebProxyAllowed returns true (thus
// allowing access)
func (gp *GemPortal) IsWebProxyAllowed(ctx *Context) bool {
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

	// Is the group EXPLICITLY listed?
	if robots != nil {
		if group := robots.FindGroup("webproxy"); group.Agent == "webproxy" {
			return group.Test(ctx.GemURL.Path)
		} else {
			return true
		}
	} else {
		return true
	}
}

// ServeGemini2HTML handles Gemini2HTML requests
func (gp *GemPortal) ServeGemini2HTML(ctx *Context) {

	if allowed := gp.IsWebProxyAllowed(ctx); !allowed {
		gp.errResp(ctx, "The host does not allow webproxies on this path", http.StatusForbidden)
		return
	}

	if values, ok := ctx.r.URL.Query()["insecure"]; ok && len(values) > 0 && values[0] == "on" {
		ctx.Insecure = true
	}

	// Append query if exists
	if len(ctx.GemInputRequest) > 0 {
		ctx.GemURL.RawQuery = ctx.GemInputRequest
	}

	client := gemini.DefaultClient
	if ctx.Insecure {
		client.Insecure = true
	}
	res, err := client.Fetch(ctx.GemURL.String())
	if err != nil {
		gp.errResp(ctx, "Could not perform gemini request: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if gemini.SimplifyStatus(res.Status) == gemini.StatusInput {
		ctx.GemInputMeta = res.Meta
		gp.executeIndexTemplate(ctx)
		return
	} else if gemini.SimplifyStatus(res.Status) == gemini.StatusRedirect {
		gp.redirectHandler(ctx, res)
		return
	} else if res.Status != gemini.StatusSuccess {
		gp.errResp(ctx, fmt.Sprintf("Gemini upstream reported '%s' (%d)", gemini.StatusText(res.Status), res.Status), http.StatusBadGateway)
		return
	}

	mediatype, _, err := mime.ParseMediaType(res.Meta)
	if err != nil {
		gp.errResp(ctx, fmt.Sprintf("Gemini upstream set bad MIME type: %s", gemini.StatusText(res.Status)), http.StatusBadGateway)
		return
	}

	// Image handling
	if strings.HasPrefix(mediatype, "image/") {
		if ctx.Raw {
			ctx.w.Header().Set("Content-Type", res.Meta)
			_ = ioLimitedCopy(ctx.w, res.Body, gp.cfg.RespMemLimitImg)
		} else {
			imgSrc := ctx.Cfg.BaseHREF + ctx.r.URL.Path + "?raw"
			ctx.GemContent = "<img src=\"" + imgSrc + "\" alt=\"" + ctx.GemURL.String() + "\">"
			gp.executeIndexTemplate(ctx)
		}
		return
	}

	html, err := gp.gemResponseToSafeHTML(ctx, res)
	if err == ErrGeminiResponseLimit {
		gp.errResp(ctx, "Response limit reached", http.StatusInternalServerError)
		return
	} else if err != nil {
		gp.errResp(ctx, "Error processing Gemini response", http.StatusInternalServerError)
		return
	}

	ctx.GemContent = html
	gp.executeIndexTemplate(ctx)
}

// redirectHandler handles Gemini redirects
func (gp *GemPortal) redirectHandler(ctx *Context, res *gemini.Response) {
	if gemini.SimplifyStatus(res.Status) != gemini.StatusRedirect {
		panic("redirectHandler is intended for 'redirect gemini.Response' only")
	}

	var newURL *url.URL

	redirectGemURL, err := gemParseURL(ctx, res.Meta)
	if err == nil {
		newURL, err = url.Parse(redirectGemURL)
	}

	if err != nil {
		gp.errResp(ctx, fmt.Sprintf("Invalid %s to '%s'", gemini.StatusText(res.Status), res.Meta), http.StatusBadGateway)
		return
	}

	ctx.GemURL = *newURL

	ctx.redirects += 1
	if ctx.redirects > gp.cfg.RedirectsLimit {
		gp.errResp(ctx, "Too many redirects", http.StatusBadGateway)
	} else {
		gp.ServeGemini2HTML(ctx)
	}
}

// errResp writes required error information into the Context and executes the index template
func (gp *GemPortal) errResp(ctx *Context, errorText string, httpStatusCode int) {
	ctx.w.WriteHeader(httpStatusCode)

	if len(errorText) > 2 {
		ctx.GemError = strings.ToUpper(errorText[0:1]) + errorText[1:]
	} else {
		ctx.GemError = errorText
	}

	ctx.GemError = bluemonday.StrictPolicy().Sanitize(ctx.GemError)

	if httpStatusCode >= 500 {
		log.Warnf("Replying with '%s' on requesting '%s': %s", http.StatusText(httpStatusCode), ctx.GemURL.String(), errorText)
	}
	gp.executeIndexTemplate(ctx)
}

// executeIndexTemplate executes the given Context with the index template.
// panics if an error occures during template execution.
func (gp *GemPortal) executeIndexTemplate(ctx *Context) {
	ctx.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := gp.indexTemplate.Execute(ctx.w, ctx)
	if err != nil {
		panic(err)
	}
}

// gemResponseToSafeHTML turns Gemtext to safe HTML and rewrites
// all Gemini URLs to hit the application server.
func (gp *GemPortal) gemResponseToSafeHTML(ctx *Context, res *gemini.Response) (string, error) {
	buf := &bytes.Buffer{}
	err := ioLimitedCopy(buf, res.Body, gp.cfg.RespMemLimit)
	if err != nil {
		return "", err
	}

	s := urlRegexp.ReplaceAllStringFunc(buf.String(), func(s string) string {
		oldURL := s

		// Strip `=> +`
		s = strings.TrimLeft(s[2:], " ")

		gemURL, err := gemParseURL(ctx, s)
		if err != nil { // Omit URL
			log.Debugf("Skipping URL from '%s'", oldURL)
			return oldURL
		}

		parsedURL, err := url.Parse(gemURL)
		if err != nil {
			log.Debugf("Skipping URL from '%s'", oldURL)
			return oldURL
		}

		newQuery := make(url.Values, 0)

		// Extract the Gemini Query (if exists)
		for query := range parsedURL.Query() {
			newQuery.Add("query", query)
			break
		}

		// Append HTTP query values (such as "unsafe=on" maybe)
		if len(ctx.r.URL.RawQuery) > 0 {
			for k, v := range ctx.r.URL.Query() {
				newQuery.Add(k, v[0])
			}
		}

		parsedURL.RawQuery = newQuery.Encode()
		gemURL = parsedURL.String()

		// Remove the scheme and leading slashes
		gemURL = strings.TrimPrefix(gemURL, "gemini://")
		gemURL = strings.TrimLeft(gemURL, "/")

		// Prepend the base HREF
		newURL := ctx.Cfg.BaseHREF + gemURL

		log.Debugf("Rewriting URL from '%s' to '%s'", oldURL, newURL)
		return "=> " + newURL
	})

	maybeUnsafeHTML := gmitohtml.Convert([]byte(s), ctx.GemURL.String())

	policy := bluemonday.UGCPolicy()
	policy.RequireNoFollowOnLinks(true)
	policy.RequireNoReferrerOnLinks(true)

	html := policy.SanitizeBytes(maybeUnsafeHTML)

	return string(html), nil
}

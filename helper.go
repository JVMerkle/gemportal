package main

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/url"
	"path"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"

	"code.rocketnine.space/tslocum/gmitohtml/pkg/gmitohtml"
	"git.sr.ht/~yotam/go-gemini"
	"github.com/microcosm-cc/bluemonday"
)

var ErrInvalidHostName = errors.New("invalid host name")
var ErrInvalidGeminiURL = errors.New("invalid Gemini URL")
var ErrInvalidGeminiScheme = errors.New("invalid Gemini URL scheme")
var ErrInvalidGeminiPort = errors.New("invalid Gemini port")
var ErrGeminiResponseLimit = errors.New("gemini response limit exceeded")
var ErrIPsProhibited = errors.New("IP addresses are prohibited")

var urlRegexp *regexp.Regexp
var hasSchemeRegexp *regexp.Regexp

func init() {
	urlRegex := `=> ([-a-zA-Z0-9()@:%_\+.~#?&//=]*)`
	urlRegexp = regexp.MustCompile(urlRegex)

	hasSchemeRegex := `[a-z]+://`
	hasSchemeRegexp = regexp.MustCompile(hasSchemeRegex)
}

// parseGeminiURL parses a Gemini URL in a git.sr.ht/~yotam/go-gemini
// conform matter.
func parseGeminiURL(ctx *ReqContext, rawURL string) (retURL *url.URL, err error) {
	// Prepend the gemini scheme
	if !strings.HasPrefix(rawURL, "gemini://") {
		rawURL = "gemini://" + rawURL
	}

	// Check if the rawURL is valid as it is
	retURL, err = url.Parse(rawURL)
	if err != nil {
		return nil, ErrInvalidGeminiURL
	}

	// Check hostname:
	// - Must contain a dot
	// - Must not be an IP address
	hostname := retURL.Hostname()
	if len(hostname) == 0 || !strings.Contains(hostname, ".") {
		return nil, ErrInvalidHostName
	} else if addr := net.ParseIP(hostname); addr != nil {
		return nil, ErrIPsProhibited
	}

	// FIXME: Enforce fully qualified domain names
	// The Gemini servers respond with BadRequest when this is enabled.
	// This is either a Gemini Client issue, or the servers are not able
	// to handle FQDN with a ending dot...
	/*if !strings.HasSuffix(hostname, ".") {
		hostname += "."
	}*/

	// Check the port
	if len(retURL.Port()) > 0 && retURL.Port() != ctx.GemDefaultPort {
		return nil, ErrInvalidGeminiPort
	}

	// Override the port anyways
	retURL.Host = hostname + ":" + ctx.GemDefaultPort

	// Fix the path
	if len(retURL.Path) == 0 {
		retURL.Path += "/"
	}

	return retURL, nil
}

// gemResponseToString reads gemtext to a length limited string (30MiB)
func gemResponseToString(ctx *ReqContext, res *gemini.Response) (string, error) {
	buf := &bytes.Buffer{}

	limit := ctx.GemRespMemLimit

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
func gemResponseToHTML(ctx *ReqContext, res *gemini.Response) (string, error) {

	s, err := gemResponseToString(ctx, res)
	if err != nil {
		return "", err
	}

	s = urlRegexp.ReplaceAllStringFunc(s, func(s string) string {
		oldURL := s

		// Strip "=> " from the RegEx match
		rawURL := s[3:]

		// Check for gemini scheme
		isAbsolute := false
		if match := hasSchemeRegexp.FindString(rawURL); len(match) >= 3 {
			scheme := match[:len(match)-3]
			if scheme == "gemini" {
				isAbsolute = true
			} else {
				// Omit URL
				return s
			}
		}

		if isAbsolute { // Absolute (with scheme)
			rawURL = rawURL[len("gemini://"):]
		} else { // Relative (without scheme)
			relDir := ""
			if !strings.HasPrefix(rawURL, "/") {
				relDir = path.Dir(ctx.GemURL.Path)
				if !strings.HasSuffix(relDir, "/") {
					relDir += "/"
				}
			}
			rawURL = ctx.GemURL.Hostname() + relDir + rawURL
		}

		// Remove the scheme from the rawURL
		// Prepend the base HREF
		newURL := ctx.BaseHREF + rawURL

		log.Debugf("Rewriting URL from '%s' to '%s'", oldURL, newURL)
		return "=> " + newURL
	})

	maybeUnsafeHTML := gmitohtml.Convert([]byte(s), ctx.GemURL.String())
	html := bluemonday.UGCPolicy().SanitizeBytes(maybeUnsafeHTML)

	return string(html), nil
}

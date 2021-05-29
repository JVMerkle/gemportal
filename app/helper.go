package app

import (
	"errors"
	"net"
	"net/url"
	"path"
	"regexp"
	"strings"
)

var ErrInvalidHostName = errors.New("invalid host name")
var ErrInvalidGeminiURL = errors.New("invalid Gemini URL")
var ErrInvalidGeminiScheme = errors.New("invalid Gemini URL scheme")
var ErrInvalidGeminiPort = errors.New("invalid Gemini port")
var ErrGeminiResponseLimit = errors.New("gemini response limit exceeded")
var ErrIPsProhibited = errors.New("IP addresses are prohibited")

var hasSchemeRegexp *regexp.Regexp

func init() {
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
	if len(retURL.Port()) > 0 && retURL.Port() != ctx.Cfg.DefaultPort {
		return nil, ErrInvalidGeminiPort
	}

	// Override the port anyways
	retURL.Host = hostname + ":" + ctx.Cfg.DefaultPort

	// Fix the path
	if len(retURL.Path) == 0 {
		retURL.Path += "/"
	}

	return retURL, nil
}

// Builds an absolute gemini URL respecting the current context
// e.g. "tata/foo.txt" on "gemini://test.com/~nana" => "gemini://test.com/~nana/tata/foo.txt"
func gemParseURL(ctx *ReqContext, gemURL string) (string, error) {
	// Check for gemini scheme
	isAbsolute := false
	if match := hasSchemeRegexp.FindString(gemURL); len(match) >= 3 {
		scheme := match[:len(match)-3]
		if scheme == "gemini" {
			isAbsolute = true
		} else {
			return "", errors.New("not a gemini URL")
		}
	}

	if !isAbsolute { // Relative (without scheme)
		relDir := ""
		if !strings.HasPrefix(gemURL, "/") {
			relDir = path.Dir(ctx.GemURL.Path)
			if !strings.HasSuffix(relDir, "/") {
				relDir += "/"
			}
		}
		gemURL = "gemini://" + ctx.GemURL.Hostname() + relDir + gemURL
	}

	return gemURL, nil
}

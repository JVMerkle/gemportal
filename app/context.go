package app

import (
	"net/http"
	"net/url"
	"strings"
)

// Context holds data required
// throughout a gemportal request.
type Context struct {
	w         http.ResponseWriter
	r         *http.Request
	redirects uint32

	Cfg Cfg

	Insecure   bool
	GemError   string
	GemURL     url.URL
	GemContent string
}

func NewContext(cfg Cfg, w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		w:   w,
		r:   r,
		Cfg: cfg,
	}
}

func (ctx *Context) PrettyPrintGemURL() string {
	hostname := ctx.GemURL.Hostname()

	// Something does not seem right...
	if len(hostname) == 0 {
		return ""
	}

	// Remove the FQDN dot if exists
	if strings.HasSuffix(hostname, ".") {
		hostname = hostname[:len(hostname)-1]
	}

	var port string
	if ctx.GemURL.Port() != ctx.Cfg.DefaultPort {
		port = ":" + ctx.GemURL.Port()
	}

	path := ctx.GemURL.Path
	if path == "/" {
		path = ""
	}

	return hostname + port + path
}

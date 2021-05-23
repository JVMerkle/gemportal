package main

import (
	"net/http"
	"net/url"
	"strings"
)

type GemContext struct {
	w               http.ResponseWriter
	r               *http.Request
	BaseHREF        string
	GemDefaultPort  string
	GemRespMemLimit int64

	DisableTLSChecks bool

	GemError string

	GemURL     url.URL
	GemContent string
}

func NewGemContext(cfg *Cfg, w http.ResponseWriter, r *http.Request) *GemContext {
	return &GemContext{
		w:               w,
		r:               r,
		BaseHREF:        cfg.BaseHREF,
		GemDefaultPort:  cfg.GemDefaultPort,
		GemRespMemLimit: cfg.GemRespMemLimit,
	}
}

func (ctx *GemContext) PrettyPrintGemURL() string {
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
	if ctx.GemURL.Port() != ctx.GemDefaultPort {
		port = ":" + ctx.GemURL.Port()
	}

	path := ctx.GemURL.Path
	if path == "/" {
		path = ""
	}

	return hostname + port + path
}

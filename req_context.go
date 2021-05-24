package main

import (
	"net/http"
	"net/url"
	"strings"
)

// ReqContext holds data required
// throughout a gemportal request.
type ReqContext struct {
	w               http.ResponseWriter
	r               *http.Request
	AppVersion      string
	AppBuildMeta    string
	BaseHREF        string
	GemDefaultPort  string
	GemRespMemLimit int64

	DisableTLSChecks bool

	GemError string

	GemURL     url.URL
	GemContent string
}

func NewReqContext(cfg *Cfg, w http.ResponseWriter, r *http.Request) *ReqContext {
	return &ReqContext{
		w:               w,
		r:               r,
		AppVersion:      cfg.AppVersion,
		AppBuildMeta:    cfg.AppBuildMeta,
		BaseHREF:        cfg.BaseHREF,
		GemDefaultPort:  cfg.GemDefaultPort,
		GemRespMemLimit: cfg.GemRespMemLimit,
	}
}

func (ctx *ReqContext) PrettyPrintGemURL() string {
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

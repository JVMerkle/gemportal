// SPDX-FileCopyrightText: 2021 Julian Merkle <me@jvmerkle.de>
//
// SPDX-License-Identifier: AGPL-3.0-only
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

	Cfg Config

	Insecure        bool
	GemError        string
	GemInputRequest string // Input data (user value)
	GemInputMeta    string // Input data description
	GemURL          url.URL
	GemContent      string
}

func NewContext(cfg Config, w http.ResponseWriter, r *http.Request) *Context {
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

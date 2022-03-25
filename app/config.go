// SPDX-FileCopyrightText: 2021-2022 Julian Merkle <me@jvmerkle.de>
//
// SPDX-License-Identifier: AGPL-3.0-only

package app

import (
	"errors"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
	"html/template"
	"strings"
)

var (
	ErrBadBaseHREF = errors.New("base HREF must begin and end with a slash")
)

// Injected with linker flags
var (
	gitHash   string
	buildTime string
)

type Config struct {
	appVersion       string        `ignored:"true"`                                 // Application version (e.g. 1.13.5)
	appBuildMeta     string        `ignored:"true"`                                 // Application build meta information (e.g. ae5f03-2021)
	AppName          string        `envconfig:"APPNAME"        default:"Gemportal"` // Application name
	AppDesc          template.HTML `envconfig:"APPDESC"`                            // Application description
	LogLevel         int           `envconfig:"LOG_LEVEL"      default:"4"`         // sirupsen/logrus log level (log.InfoLevel)
	HTTPPort         string        `envconfig:"HTTP_PORT"      default:"8080"`      // HTTP server port
	BaseHREF         string        `envconfig:"BASE_HREF"      default:"/"`         // Base HREF for the application (e.g. / or /gemportal/)
	DefaultPort      string        `envconfig:"DEFAULT_PORT"   default:"1965"`      // Default Gemini port
	RespMemLimit     int64         `envconfig:"RESP_MEM_LIMIT" default:"31457280"`  // Gemini response limit in bytes (30MiB)
	RedirectsLimit   uint32        `envconfig:"MAX_REDIRECTS"  default:"3"`         // Maximum gemini redirects to follow
	InputPlaceholder string        `envconfig:"INPUT_PLACEHOLDER"  default:""`      // Placeholder for the HTML input form
}

func (c *Config) GetAppVersion() string {
	return c.appVersion
}

func (c *Config) GetAppBuildMeta() string {
	return c.appBuildMeta
}

// GetConfig loads and checks the application configuration
func GetConfig(appVersion string) (*Config, error) {
	var cfg Config

	// Load environment variables from a ".env" file, if exists
	_ = godotenv.Load()

	err := envconfig.Process("GEM", &cfg)
	if err != nil {
		return nil, err
	}

	if len(gitHash) == 0 || len(buildTime) == 0 {
		log.Warn("Link time definitions missing (GitHash/Buildtime)")
	}

	if len(cfg.AppDesc) == 0 {
		cfg.AppDesc = template.HTML("<b>Simple</b> Gemini HTTP portal for port " + cfg.DefaultPort + ".")
	}

	buildMeta := []string{gitHash, buildTime}

	cfg.appVersion = appVersion

	for i := 0; i < len(buildMeta); i++ {
		cfg.appBuildMeta += buildMeta[i]
		cfg.appBuildMeta += "-"
	}

	// Remove the last dash "-"
	if len(cfg.appBuildMeta) > 0 {
		cfg.appBuildMeta = cfg.appBuildMeta[:len(cfg.appBuildMeta)-2]
	}

	// The base HREF has to be pre- and suffixed by a slash
	if !strings.HasPrefix(cfg.BaseHREF, "/") || !strings.HasSuffix(cfg.BaseHREF, "/") {
		return nil, ErrBadBaseHREF
	}

	return &cfg, nil
}

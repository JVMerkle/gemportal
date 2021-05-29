package app

import (
	"errors"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

var (
	ErrLinkerDefsMissing = errors.New("link time definitions missing (GitHash/Buildtime)")
	ErrBadBaseHREF       = errors.New("BASE_HREF must begin and end with a slash")
)

// Injected with linker flags
var (
	gitHash   string
	buildTime string
)

type Cfg struct {
	AppVersion     string `ignored:"true"`                                    // Application version (e.g. 1.13.5)
	AppBuildMeta   string `ignored:"true"`                                    // Application build meta information (e.g. ae5f03-2021)
	LogLevel       int    `envconfig:"LOGLEVEL" default:"4"`                  // sirupsen/logrus log level (log.InfoLevel)
	HTTPPort       string `envconfig:"HTTP_PORT" default:"8080"`              // HTTP server port
	BaseHREF       string `envconfig:"BASE_HREF" default:"/"`                 // Base HREF for the application (e.g. / or /gemportal/)
	DefaultPort    string `envconfig:"GEM_DEFAULT_PORT" default:"1965"`       // Default Gemini port
	RespMemLimit   int64  `envconfig:"GEM_RESP_MEM_LIMIT" default:"31457280"` // Gemini response limit in bytes (30MiB)
	RedirectsLimit uint32 `envconfig:"GEM_MAX_REDIRECTS" default:"3"`         // Maximum gemini redirects to follow
}

// GetConfig loads and checks the application configuration
func GetConfig(appVersion string) (*Cfg, error) {
	var cfg Cfg

	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}

	if len(gitHash) == 0 || len(buildTime) == 0 {
		return nil, ErrLinkerDefsMissing
	}
	cfg.AppVersion = appVersion
	cfg.AppBuildMeta = gitHash + "-" + buildTime

	// The base HREF has to be pre- and suffixed by a slash
	if !strings.HasPrefix(cfg.BaseHREF, "/") || !strings.HasSuffix(cfg.BaseHREF, "/") {
		return nil, ErrBadBaseHREF
	}

	return &cfg, nil
}

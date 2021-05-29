package app

import (
	"errors"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

var (
	ErrLinkerDefsMissing = errors.New("link time definitions missing (GitHash/Buildtime)")
	ErrBadBaseHREF       = errors.New("base HREF must begin and end with a slash")
)

// Injected with linker flags
var (
	gitHash   string
	buildTime string
)

type Cfg struct {
	appVersion     string `ignored:"true"`                                // Application version (e.g. 1.13.5)
	appBuildMeta   string `ignored:"true"`                                // Application build meta information (e.g. ae5f03-2021)
	LogLevel       int    `envconfig:"LOG_LEVEL"      default:"4"`        // sirupsen/logrus log level (log.InfoLevel)
	HTTPPort       string `envconfig:"HTTP_PORT"      default:"8080"`     // HTTP server port
	BaseHREF       string `envconfig:"BASE_HREF"      default:"/"`        // Base HREF for the application (e.g. / or /gemportal/)
	DefaultPort    string `envconfig:"DEFAULT_PORT"   default:"1965"`     // Default Gemini port
	RespMemLimit   int64  `envconfig:"RESP_MEM_LIMIT" default:"31457280"` // Gemini response limit in bytes (30MiB)
	RedirectsLimit uint32 `envconfig:"MAX_REDIRECTS"  default:"3"`        // Maximum gemini redirects to follow
}

func (c *Cfg) GetAppVersion() string {
	return c.appVersion
}

func (c *Cfg) GetAppBuildMeta() string {
	return c.appBuildMeta
}

// GetConfig loads and checks the application configuration
func GetConfig(appVersion string) (*Cfg, error) {
	var cfg Cfg

	err := envconfig.Process("GEM", &cfg)
	if err != nil {
		return nil, err
	}

	if len(gitHash) == 0 || len(buildTime) == 0 {
		return nil, ErrLinkerDefsMissing
	}
	cfg.appVersion = appVersion
	cfg.appBuildMeta = gitHash + "-" + buildTime

	// The base HREF has to be pre- and suffixed by a slash
	if !strings.HasPrefix(cfg.BaseHREF, "/") || !strings.HasSuffix(cfg.BaseHREF, "/") {
		return nil, ErrBadBaseHREF
	}

	return &cfg, nil
}

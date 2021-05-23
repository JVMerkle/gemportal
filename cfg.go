package main

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

var (
	ErrBadLogLevel        = errors.New("bad LOGLEVEL (must be numeric)")
	ErrBadHTTPPort        = errors.New("bad HTTP_PORT (must be numeric)")
	ErrBadBaseHREF        = errors.New("BASE_HREF must begin and end with a slash")
	ErrBadGemDefaultPort  = errors.New("GEM_DEFAULT_PORT is not an integer")
	ErrBadGemRespMemLimit = errors.New("GEM_RESP_MEM_LIMIT is not an integer")
)

type Cfg struct {
	LogLevel        int    // sirupsen/logrus log level
	HTTPPort        string // HTTP server port
	BaseHREF        string // Base HREF for the application (e.g. / or /gemportal/)
	GemDefaultPort  string // Default Gemini port
	GemRespMemLimit int64  // Gemini response limit in bytes
}

// GetConfig loads and checks the application configuration
func GetConfig() (*Cfg, error) {
	var cfg Cfg

	_ = godotenv.Load()

	if logLevel := os.Getenv("LOGLEVEL"); len(logLevel) == 0 {
		cfg.LogLevel = int(log.InfoLevel)
	} else if logLevelInt, err := strconv.Atoi(logLevel); err != nil {
		return nil, ErrBadLogLevel
	} else {
		cfg.LogLevel = logLevelInt
	}

	if httpPort := os.Getenv("HTTP_PORT"); len(httpPort) == 0 {
		cfg.HTTPPort = "8080"
	} else if !isInteger(httpPort) {
		return nil, ErrBadHTTPPort
	} else {
		cfg.HTTPPort = httpPort
	}

	if baseHREF := os.Getenv("BASE_HREF"); len(baseHREF) == 0 {
		cfg.BaseHREF = "/"
	} else if baseHREF[0] != '/' || baseHREF[len(baseHREF)-1] != '/' {
		return nil, ErrBadBaseHREF
	} else {
		cfg.BaseHREF = baseHREF
	}

	if gemDefaultPort := os.Getenv("GEM_DEFAULT_PORT"); len(gemDefaultPort) == 0 {
		cfg.GemDefaultPort = "1965"
	} else if !isInteger(gemDefaultPort) {
		return nil, ErrBadGemDefaultPort
	} else {
		cfg.GemDefaultPort = gemDefaultPort
	}

	if value := os.Getenv("GEM_RESP_MEM_LIMIT"); len(value) == 0 {
		cfg.GemRespMemLimit = int64(30 * 1024 * 1024) // 30 MiB
	} else if num, err := strconv.ParseInt(value, 10, 64); err != nil {
		return nil, ErrBadGemRespMemLimit
	} else {
		cfg.GemRespMemLimit = num
	}

	return &cfg, nil
}

// isInteger checks if a string is an integer
func isInteger(s string) bool {
	if _, err := strconv.Atoi(s); err != nil {
		return false
	} else {
		return true
	}
}

package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	SourceVault      string
	TargetVaults     []string
	Include          string // comma-separated globs
	Exclude          string // comma-separated globs
	DryRun           bool
	Concurrency      int
	OverrideDisabled bool
	LatestOnly       bool
	NoBrowserAuth    bool
}

func FromEnv() Config {
	c := Config{
		SourceVault:      os.Getenv("SOURCE_VAULT"),
		TargetVaults:     splitAndTrim(os.Getenv("TARGET_VAULTS")),
		Include:          os.Getenv("INCLUDE_PATTERNS"),
		Exclude:          os.Getenv("EXCLUDE_PATTERNS"),
		DryRun:           getBool("DRY_RUN", false),
		Concurrency:      getInt("CONCURRENCY", 8),
		OverrideDisabled: getBool("OVERRIDE_DISABLED", false),
		LatestOnly:       getBool("LATEST_ONLY", true),
		NoBrowserAuth:    os.Getenv("NO_BROWSER_AUTH") == "true",
	}
	if v := os.Getenv("TARGET_VAULTS"); v != "" {
		for _, t := range strings.Split(v, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				c.TargetVaults = append(c.TargetVaults, t)
			}
		}
	}
	return c
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		b, err := strconv.ParseBool(v)
		if err == nil {
			return b
		}
	}
	return def
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		i, err := strconv.Atoi(v)
		if err == nil {
			return i
		}
	}
	return def
}

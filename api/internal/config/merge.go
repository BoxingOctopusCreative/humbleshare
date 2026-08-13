package config

import (
	"os"
	"strconv"
	"strings"
)

func infLookup(inf map[string]string, envKey string) string {
	if inf == nil {
		return ""
	}
	return strings.TrimSpace(inf[envKey])
}

func mergeString(envKey, cli, file string, inf map[string]string, def string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	if strings.TrimSpace(cli) != "" {
		return strings.TrimSpace(cli)
	}
	if strings.TrimSpace(file) != "" {
		return strings.TrimSpace(file)
	}
	if v := infLookup(inf, envKey); v != "" {
		return v
	}
	return def
}

func mergeStringSlice(envKey, cliCSV string, fileSlice []string, inf map[string]string, def []string) []string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return splitCSV(v)
	}
	if strings.TrimSpace(cliCSV) != "" {
		return splitCSV(cliCSV)
	}
	if len(fileSlice) > 0 {
		out := make([]string, 0, len(fileSlice))
		for _, s := range fileSlice {
			s = strings.TrimSpace(s)
			if s != "" {
				out = append(out, s)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if v := infLookup(inf, envKey); v != "" {
		return splitCSV(v)
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envBoolSet(key string) (value bool, set bool) {
	s := os.Getenv(key)
	if s == "" {
		return false, false
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return s == "1" || strings.EqualFold(s, "yes"), true
	}
	return b, true
}

func parseBoolString(s string) (value bool, set bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return false, false
	}
	b, err := strconv.ParseBool(s)
	if err != nil {
		return s == "1" || strings.EqualFold(s, "yes"), true
	}
	return b, true
}

func mergeBool(envKey, cliStr string, fileVal *bool, inf map[string]string, def bool) bool {
	if v, ok := envBoolSet(envKey); ok {
		return v
	}
	if strings.TrimSpace(cliStr) != "" {
		b, err := strconv.ParseBool(strings.TrimSpace(cliStr))
		if err != nil {
			s := strings.TrimSpace(cliStr)
			return s == "1" || strings.EqualFold(s, "yes")
		}
		return b
	}
	if fileVal != nil {
		return *fileVal
	}
	if v, ok := parseBoolString(infLookup(inf, envKey)); ok {
		return v
	}
	return def
}

func mergePositiveInt(envKey string, fileVal int, inf map[string]string, def int) int {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	if fileVal > 0 {
		return fileVal
	}
	if v := infLookup(inf, envKey); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func mergePositiveFloat(envKey string, fileVal float64, inf map[string]string, def float64) float64 {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	if fileVal > 0 {
		return fileVal
	}
	if v := infLookup(inf, envKey); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return def
}

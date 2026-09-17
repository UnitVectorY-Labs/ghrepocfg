package config

import (
	"bytes"
	"encoding/json"
	"strings"
)

// WithUnavailable returns a copy with unknown paths omitted, or restored from
// fallback during a scoped refresh. Path components preserve literal map keys.
func WithUnavailable(c, fallback *Config, paths [][]string) *Config {
	toMap := func(c *Config) map[string]any {
		m := map[string]any{}
		if c != nil {
			b, _ := json.Marshal(c)
			decoder := json.NewDecoder(bytes.NewReader(b))
			decoder.UseNumber()
			_ = decoder.Decode(&m)
		}
		return m
	}
	out, base := toMap(c), toMap(fallback)
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		var value any = base
		present := fallback != nil
		for _, key := range path {
			m, ok := value.(map[string]any)
			if !ok {
				present = false
				break
			}
			key = actualKey(m, key)
			value, ok = m[key]
			if !ok {
				present = false
				break
			}
		}
		setPath(out, path, value, present)
	}
	b, _ := json.Marshal(out)
	var result Config
	_ = json.Unmarshal(b, &result)
	return &result
}

func setPath(m map[string]any, path []string, value any, present bool) {
	key := actualKey(m, path[0])
	if len(path) == 1 {
		if present {
			m[key] = value
		} else {
			delete(m, key)
		}
		return
	}
	child, ok := m[key].(map[string]any)
	if !ok {
		if !present {
			return
		}
		child = map[string]any{}
		m[key] = child
	}
	setPath(child, path[1:], value, present)
}

func actualKey(m map[string]any, key string) string {
	if _, ok := m[key]; ok {
		return key
	}
	for k := range m {
		if strings.EqualFold(k, key) {
			return k
		}
	}
	return key
}

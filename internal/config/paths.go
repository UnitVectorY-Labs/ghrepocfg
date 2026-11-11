package config

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
)

// WithUnavailable returns a copy with unknown paths omitted, or restored from
// fallback during a scoped refresh. Path components preserve literal map keys.
func WithUnavailable(c, fallback *Config, paths [][]string) *Config {
	toMap := func(c *Config) map[string]jsontext.Value {
		m := map[string]jsontext.Value{}
		if c != nil {
			b, _ := json.Marshal(c)
			_ = json.Unmarshal(b, &m)
		}
		return m
	}
	out, base := toMap(c), toMap(fallback)
	for _, path := range paths {
		if len(path) == 0 {
			continue
		}
		value, _ := json.Marshal(base)
		present := fallback != nil
		for _, key := range path {
			var m map[string]jsontext.Value
			if json.Unmarshal(value, &m) != nil || m == nil {
				present = false
				break
			}
			key = actualKey(m, key)
			var ok bool
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

func setPath(m map[string]jsontext.Value, path []string, value jsontext.Value, present bool) {
	key := actualKey(m, path[0])
	if len(path) == 1 {
		if present {
			m[key] = value
		} else {
			delete(m, key)
		}
		return
	}
	var child map[string]jsontext.Value
	if json.Unmarshal(m[key], &child) != nil || child == nil {
		if !present {
			return
		}
		child = map[string]jsontext.Value{}
	}
	setPath(child, path[1:], value, present)
	m[key], _ = json.Marshal(child)
}

func actualKey(m map[string]jsontext.Value, key string) string {
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

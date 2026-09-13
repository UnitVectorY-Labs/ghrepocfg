package config

// This file contains the canonical, local representation used when comparing
// configurations.  It deliberately retains pointers' presence: a nil field
// means unmanaged, whereas a non-nil pointer to an empty map or slice is an
// explicit authoritative value.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// SemanticTree returns a JSON-compatible tree of the managed portions of c.
// Maps are represented by string-keyed maps and values are normalized using
// the same domain conventions used by reconciliation (case-insensitive
// variable names and unordered collection fields).
func SemanticTree(c *Config) (map[string]any, error) {
	if c == nil {
		return nil, fmt.Errorf("nil configuration")
	}
	v := semanticValue(reflect.ValueOf(c), nil, false)
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("configuration did not produce an object")
	}
	return m, nil
}

func semanticValue(v reflect.Value, path []string, unordered bool) any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		return semanticValue(v.Elem(), path, unordered)
	}
	if v.Type() == reflect.TypeFor[CustomPropertyValue]() {
		return semanticValue(reflect.ValueOf(v.Interface().(CustomPropertyValue).Value), path, true)
	}
	switch v.Kind() {
	case reflect.Struct:
		m := map[string]any{}
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			name := tag[0]
			if name == "" || name == "-" {
				continue
			}
			fv := v.Field(i)
			if fv.Kind() == reflect.Pointer && fv.IsNil() {
				continue
			}
			// Non-pointer slices/maps are value fields in the model. Nil and empty
			// have the same literal semantics for these fields, so retain both as
			// an empty collection in the semantic tree.
			if fv.Kind() != reflect.Pointer && fv.Kind() != reflect.Slice && fv.Kind() != reflect.Map && strings.Contains(","+f.Tag.Get("json")+",", ",omitempty,") && fv.IsZero() {
				continue
			}
			child := append(append([]string(nil), path...), name)
			x := semanticValue(fv, child, unorderedField(t, name))
			if t == reflect.TypeFor[Label]() && name == "color" {
				if color, ok := x.(string); ok {
					x = strings.ToLower(color)
				}
			}
			if x != nil || (fv.Kind() == reflect.Pointer && !fv.IsNil()) || fv.Kind() == reflect.Slice || fv.Kind() == reflect.Map {
				m[name] = x
			}
		}
		if t == reflect.TypeFor[Ruleset]() && v.Interface().(Ruleset).Target == "" {
			m["target"] = "branch"
		}
		if t == reflect.TypeFor[BypassActor]() && v.Interface().(BypassActor).BypassMode == "" {
			m["bypass_mode"] = "always"
		}
		if t == reflect.TypeFor[DeployKey]() {
			parts := strings.Fields(v.Interface().(DeployKey).Key)
			if len(parts) >= 2 {
				m["key"] = strings.Join(parts[:2], " ")
			}
		}
		if t == reflect.TypeFor[Rule]() && ruleUpdateDefaults(v) {
			m["parameters"] = map[string]any{"update_allows_fetch_and_merge": false}
		}
		return m
	case reflect.Map:
		m := map[string]any{}
		if v.IsNil() {
			return m
		}
		for _, k := range v.MapKeys() {
			name := CanonicalKey(path, fmt.Sprint(k.Interface()))
			m[name] = semanticValue(v.MapIndex(k), append(append([]string(nil), path...), name), false)
		}
		return m
	case reflect.Slice, reflect.Array:
		a := make([]any, v.Len())
		for i := range a {
			a[i] = semanticValue(v.Index(i), path, false)
		}
		if unordered {
			sort.SliceStable(a, func(i, j int) bool {
				x, _ := json.Marshal(a[i])
				y, _ := json.Marshal(a[j])
				return string(x) < string(y)
			})
		}
		return a
	default:
		return v.Interface()
	}
}

// CanonicalKey applies case-insensitive identity rules to map keys. Path contains
// decoded configuration path segments, so names containing punctuation are safe.
func CanonicalKey(path []string, key string) string {
	if len(path) == 1 {
		switch path[0] {
		case "collaborators", "teams", "labels", "environments":
			return strings.ToLower(key)
		}
	}
	if (len(path) == 2 && path[0] == "actions" && path[1] == "variables") || (len(path) == 3 && path[0] == "environments" && path[2] == "variables") {
		return strings.ToUpper(key)
	}
	return key
}

func ruleUpdateDefaults(v reflect.Value) bool {
	r, ok := v.Interface().(Rule)
	if !ok || r.Type != "update" {
		return false
	}
	return r.Parameters == nil || r.Parameters.UpdateAllowsFetchAndMerge == nil || !*r.Parameters.UpdateAllowsFetchAndMerge
}

// Array ordering is domain-specific. OIDC claim keys deliberately remain ordered.
// Rules and rule selections are unordered, including structured members.
func unorderedField(parent reflect.Type, field string) bool {
	switch parent {
	case reflect.TypeFor[RepositorySettings]():
		return field == "topics"
	case reflect.TypeFor[SelectedActions]():
		return field == "patterns_allowed"
	case reflect.TypeFor[CodeScanningSetup]():
		return field == "languages"
	case reflect.TypeFor[Environment]():
		return field == "reviewers" || field == "deployment_branch_patterns" || field == "deployment_tag_patterns"
	case reflect.TypeFor[DelegatedBypassOptions]():
		return field == "reviewers"
	case reflect.TypeFor[Ruleset]():
		return field == "rules" || field == "bypass_actors"
	case reflect.TypeFor[RefNameCondition]():
		return field == "include" || field == "exclude"
	case reflect.TypeFor[RuleParameters](), reflect.TypeFor[DismissalRestriction](), reflect.TypeFor[RequiredReviewer]():
		return true
	}
	return false
}

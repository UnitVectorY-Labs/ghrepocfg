// Package policy materializes literal configuration layers and compares their meaning.
// It performs no network, authentication, repository selection, or Git operations.
package policy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/UnitVectorY-Labs/ghrepocfg/internal/config"
	"gopkg.in/yaml.v3"
)

type Lock struct {
	Path string `yaml:"path"`
	Mode string `yaml:"mode"`
}
type Constraints struct {
	Locks []Lock `yaml:"locks"`
}
type Layer struct {
	Name        string
	Config      []byte
	Constraints []byte
}
type established struct {
	Lock
	value any
	layer string
}

func ParseConstraints(data []byte) (*Constraints, error) {
	d := yaml.NewDecoder(bytes.NewReader(data))
	d.KnownFields(true)
	var decoded struct {
		Locks []struct {
			Path *string `yaml:"path"`
			Mode string  `yaml:"mode"`
		} `yaml:"locks"`
	}
	if err := d.Decode(&decoded); err != nil {
		return nil, fmt.Errorf("invalid constraints: %w", err)
	}
	var extra any
	if err := d.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid constraints: expected one YAML document")
	}
	// Require an explicit mapping, including for an empty policy.
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	if len(node.Content) != 1 || node.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("invalid constraints: expected a mapping with locks")
	}
	var c Constraints
	for _, item := range decoded.Locks {
		if item.Path == nil {
			return nil, errors.New("invalid constraint: path is required (use an explicit empty string for the root)")
		}
		l := Lock{Path: *item.Path, Mode: item.Mode}
		c.Locks = append(c.Locks, l)
		if _, err := pointer(l.Path); err != nil {
			return nil, err
		}
		if l.Mode != "exact" && l.Mode != "contains" {
			return nil, fmt.Errorf("invalid constraint at %q: mode must be exact or contains", l.Path)
		}
	}
	return &c, nil
}

// Resolve validates every literal layer and every candidate before emitting anything.
func Resolve(layers []Layer) ([]byte, error) {
	if len(layers) == 0 {
		return nil, errors.New("resolve requires at least one layer")
	}
	tree := map[string]any{}
	var locks []established
	var result *config.Config
	for _, layer := range layers {
		if _, err := config.Parse(layer.Config); err != nil {
			return nil, fmt.Errorf("layer %s: %w", layer.Name, err)
		}
		var document yaml.Node
		if err := yaml.Unmarshal(layer.Config, &document); err != nil {
			return nil, err
		}
		authored, err := authoredValue(document.Content[0], reflect.TypeOf(config.Config{}))
		if err != nil {
			return nil, fmt.Errorf("layer %s: %w", layer.Name, err)
		}
		values := authored.(map[string]any)
		prepare(values, reflect.TypeOf(config.Config{}), nil)
		merge(tree, values)
		b, err := yaml.Marshal(tree)
		if err != nil {
			return nil, err
		}
		result, err = config.Parse(b)
		if err != nil {
			return nil, fmt.Errorf("after layer %s: %w", layer.Name, err)
		}
		semantic, err := config.SemanticTree(result)
		if err != nil {
			return nil, err
		}
		for _, l := range locks {
			value, ok := lookup(semantic, l.Path)
			valid := ok && reflect.DeepEqual(l.value, value)
			if l.Mode == "contains" {
				valid = ok && contains(value, l.value)
			}
			if !valid {
				return nil, fmt.Errorf("constraint violation at %s: %s lock established by layer %s requires %s; layer %s produces %s (present: %t)", l.Path, l.Mode, l.layer, display(l.value), layer.Name, display(value), ok)
			}
		}
		if layer.Constraints != nil {
			constraints, err := ParseConstraints(layer.Constraints)
			if err != nil {
				return nil, fmt.Errorf("layer %s: %w", layer.Name, err)
			}
			for _, l := range constraints.Locks {
				value, ok := lookup(semantic, l.Path)
				if !ok {
					return nil, fmt.Errorf("layer %s: constraint path %q does not resolve", layer.Name, l.Path)
				}
				if l.Mode == "contains" {
					if _, ok := value.([]any); !ok {
						return nil, fmt.Errorf("layer %s: contains constraint at %s requires an array", layer.Name, l.Path)
					}
				}
				locks = append(locks, established{l, value, layer.Name})
			}
		}
	}
	canonical, err := config.SemanticTree(result)
	if err != nil {
		return nil, err
	}
	b, err := yaml.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	result, err = config.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("resolved configuration: %w", err)
	}
	return config.Marshal(result)
}

// authoredValue retains authored field presence while decoding mapping keys as
// strings just like Config. Decoding node mappings also honors YAML merge keys.
func authoredValue(node *yaml.Node, typ reflect.Type) (any, error) {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!null" {
		return nil, nil
	}
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if node.Kind == yaml.AliasNode {
		return authoredValue(node.Alias, typ)
	}
	switch node.Kind {
	case yaml.MappingNode:
		var nodes map[string]yaml.Node
		if err := node.Decode(&nodes); err != nil {
			return nil, err
		}
		result := make(map[string]any, len(nodes))
		for k, n := range nodes {
			v, err := authoredValue(&n, fieldType(typ, k))
			if err != nil {
				return nil, err
			}
			result[k] = v
		}
		return result, nil
	case yaml.SequenceNode:
		result := make([]any, len(node.Content))
		for i, n := range node.Content {
			v, err := authoredValue(n, sequenceElement(typ))
			if err != nil {
				return nil, err
			}
			result[i] = v
		}
		return result, nil
	default:
		switch typ.Kind() {
		case reflect.String, reflect.Bool, reflect.Int, reflect.Int64:
			result := reflect.New(typ)
			err := node.Decode(result.Interface())
			return result.Elem().Interface(), err
		}
		var result any
		err := node.Decode(&result)
		return result, err
	}
}

func fieldType(typ reflect.Type, key string) reflect.Type {
	if typ.Kind() == reflect.Map {
		return typ.Elem()
	}
	if typ.Kind() == reflect.Struct {
		for i := 0; i < typ.NumField(); i++ {
			if strings.Split(typ.Field(i).Tag.Get("yaml"), ",")[0] == key {
				return typ.Field(i).Type
			}
		}
	}
	return reflect.TypeOf((*any)(nil)).Elem()
}
func sequenceElement(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array {
		return typ.Elem()
	}
	return reflect.TypeOf((*any)(nil)).Elem()
}

// Prepare authored mappings without introducing omitted fields or model defaults.
// Only pointer nulls mean omission; null non-pointer fields retain their literal
// zero-value semantics. Canonical identities let case variants override by key.
func prepare(m map[string]any, typ reflect.Type, path []string) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	for k, v := range m {
		field := fieldType(typ, k)
		if v == nil && field.Kind() == reflect.Pointer {
			delete(m, k)
			continue
		}
		key := config.CanonicalKey(path, k)
		if child, ok := v.(map[string]any); ok {
			prepare(child, field, append(append([]string(nil), path...), key))
		}
		if key != k {
			delete(m, k)
			m[key] = v
		}
	}
}
func merge(dst, src map[string]any) {
	for k, v := range src {
		dm, dok := dst[k].(map[string]any)
		sm, sok := v.(map[string]any)
		if dok && sok {
			merge(dm, sm)
		} else {
			dst[k] = v
		}
	}
}
func pointer(path string) ([]string, error) {
	if path == "" {
		return nil, nil
	}
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("invalid JSON Pointer %q: expected leading /", path)
	}
	parts := strings.Split(path[1:], "/")
	for i, p := range parts {
		for j := 0; j < len(p); j++ {
			if p[j] == '~' {
				if j+1 == len(p) || (p[j+1] != '0' && p[j+1] != '1') {
					return nil, fmt.Errorf("invalid JSON Pointer %q: invalid ~ escape", path)
				}
				j++
			}
		}
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(p, "~1", "/"), "~0", "~")
	}
	return parts, nil
}
func escape(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1") }
func lookup(tree any, path string) (any, bool) {
	parts, err := pointer(path)
	if err != nil {
		return nil, false
	}
	var traversed []string
	for _, p := range parts {
		switch v := tree.(type) {
		case map[string]any:
			p = config.CanonicalKey(traversed, p)
			var ok bool
			tree, ok = v[p]
			if !ok {
				return nil, false
			}
		case []any:
			i, err := strconv.Atoi(p)
			if err != nil || i < 0 || i >= len(v) || strconv.Itoa(i) != p {
				return nil, false
			}
			tree = v[i]
		default:
			return nil, false
		}
		traversed = append(traversed, p)
	}
	return tree, true
}
func contains(value, required any) bool {
	values, ok := value.([]any)
	if !ok {
		return false
	}
	requirements, ok := required.([]any)
	if !ok {
		return false
	}
	used := make([]bool, len(values))
	for _, r := range requirements {
		found := false
		for i, v := range values {
			if !used[i] && reflect.DeepEqual(r, v) {
				used[i] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
func display(v any) string { b, _ := json.Marshal(v); return string(b) }

// Change uses JSON Pointers. RawMessage preserves explicit null separately from absence.
type Change struct {
	Path      string          `json:"path"`
	Operation string          `json:"operation"`
	Before    json.RawMessage `json:"before,omitempty"`
	After     json.RawMessage `json:"after,omitempty"`
}
type Diff struct {
	Changed bool     `json:"changed"`
	Changes []Change `json:"changes"`
}

func Compare(a, b *config.Config) (Diff, error) {
	result := Diff{Changes: []Change{}}
	before, err := config.SemanticTree(a)
	if err != nil {
		return result, err
	}
	after, err := config.SemanticTree(b)
	if err != nil {
		return result, err
	}
	walk("", before, true, after, true, &result.Changes)
	result.Changed = len(result.Changes) > 0
	return result, nil
}
func walk(path string, a any, ap bool, b any, bp bool, out *[]Change) {
	if ap && bp && reflect.DeepEqual(a, b) {
		return
	}
	am, aok := a.(map[string]any)
	bm, bok := b.(map[string]any)
	if ap && bp && aok && bok {
		keys := map[string]bool{}
		for k := range am {
			keys[k] = true
		}
		for k := range bm {
			keys[k] = true
		}
		ordered := make([]string, 0, len(keys))
		for k := range keys {
			ordered = append(ordered, k)
		}
		sort.Strings(ordered)
		for _, k := range ordered {
			av, ao := am[k]
			bv, bo := bm[k]
			walk(path+"/"+escape(k), av, ao, bv, bo, out)
		}
		return
	}
	c := Change{Path: path, Operation: "modify"}
	if ap {
		c.Before, _ = json.Marshal(a)
	} else {
		c.Operation = "add"
	}
	if bp {
		c.After, _ = json.Marshal(b)
	} else {
		c.Operation = "remove"
	}
	*out = append(*out, c)
}

package cmdutil

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseStringFields parses repeated KEY=VALUE flags into a nested map.
func ParseStringFields(pairs []string) (map[string]any, error) {
	out := map[string]any{}
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid field %q: expected KEY=VALUE", pair)
		}
		if err := setPath(out, key, value); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ParseRawFields parses repeated KEY=JSON flags, falling back to strings when
// the value is not valid JSON.
func ParseRawFields(pairs []string) (map[string]any, error) {
	out := map[string]any{}
	for _, pair := range pairs {
		key, value, ok := strings.Cut(pair, "=")
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid field %q: expected KEY=VALUE", pair)
		}
		var parsed any
		if err := json.Unmarshal([]byte(value), &parsed); err != nil {
			parsed = value
		}
		if err := setPath(out, key, parsed); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func setPath(out map[string]any, key string, value any) error {
	parts := strings.Split(key, ".")
	current := out
	for i, part := range parts {
		if part == "" {
			return fmt.Errorf("invalid field name %q", key)
		}
		if i == len(parts)-1 {
			current[part] = value
			return nil
		}
		next, ok := current[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[part] = next
		}
		current = next
	}
	return nil
}

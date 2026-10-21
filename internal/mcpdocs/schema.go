package mcpdocs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// schema is the part of a JSON Schema the reference shows. Nested schemas
// stay raw, so their properties can be read in source order.
type schema struct {
	Type             json.RawMessage   `json:"type"`
	Format           string            `json:"format"`
	Enum             []json.RawMessage `json:"enum"`
	Const            json.RawMessage   `json:"const"`
	Pattern          string            `json:"pattern"`
	Minimum          *float64          `json:"minimum"`
	Maximum          *float64          `json:"maximum"`
	ExclusiveMinimum json.RawMessage   `json:"exclusiveMinimum"` // a number, or a bool before draft 6
	ExclusiveMaximum json.RawMessage   `json:"exclusiveMaximum"`
	MinLength        *int              `json:"minLength"`
	MaxLength        *int              `json:"maxLength"`
	MinItems         *int              `json:"minItems"`
	MaxItems         *int              `json:"maxItems"`
	Default          json.RawMessage   `json:"default"`
	Description      string            `json:"description"`
	Items            json.RawMessage   `json:"items"`
	Properties       json.RawMessage   `json:"properties"`
	Required         []string          `json:"required"`
	AnyOf            []json.RawMessage `json:"anyOf"`
	OneOf            []json.RawMessage `json:"oneOf"`
	Ref              string            `json:"$ref"`
}

// property is one entry of a schema's properties, in source order.
type property struct {
	Name     string
	Schema   schema
	Required bool
}

func parseSchema(raw json.RawMessage) (schema, error) {
	var s schema
	if len(raw) == 0 || string(raw) == "null" {
		return s, nil
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, fmt.Errorf("parsing schema: %w", err)
	}
	return s, nil
}

// properties returns the schema's properties in the order the server sent
// them; decoding into a map would sort them.
func (s schema) properties() ([]property, error) {
	if len(s.Properties) == 0 || string(s.Properties) == "null" {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(s.Properties))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return nil, fmt.Errorf("properties is not an object")
	}
	var props []property
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, fmt.Errorf("reading properties: %w", err)
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, fmt.Errorf("reading property %v: %w", tok, err)
		}
		name, _ := tok.(string)
		sub, err := parseSchema(raw)
		if err != nil {
			return nil, fmt.Errorf("property %s: %w", name, err)
		}
		props = append(props, property{Name: name, Schema: sub, Required: slices.Contains(s.Required, name)})
	}
	return props, nil
}

// types returns "type" as a list; it may be a string or an array.
func (s schema) types() []string {
	var one string
	if json.Unmarshal(s.Type, &one) == nil && one != "" {
		return []string{one}
	}
	var many []string
	_ = json.Unmarshal(s.Type, &many)
	return many
}

// baseType names the type, e.g. "string", "array of integer", "string | null".
func (s schema) baseType() string {
	var parts []string
	for _, t := range s.types() {
		if t == "array" {
			if items, err := parseSchema(s.Items); err == nil && items.baseType() != "any" {
				t = "array of " + items.baseType()
			}
		}
		parts = append(parts, t)
	}
	for _, alt := range append(slices.Clone(s.AnyOf), s.OneOf...) {
		if sub, err := parseSchema(alt); err == nil {
			parts = append(parts, sub.baseType())
		}
	}
	switch {
	case len(parts) > 0:
		return strings.Join(parts, " | ")
	case s.Ref != "":
		return s.Ref[strings.LastIndex(s.Ref, "/")+1:]
	case len(s.Const) > 0:
		return "constant"
	case len(s.Properties) > 0:
		return "object"
	}
	return "any"
}

// typeText is the type with its constraints, as shown in the Type column.
func (s schema) typeText() string {
	var details []string
	if s.Format != "" {
		details = append(details, s.Format)
	}
	if len(s.Enum) > 0 {
		values := make([]string, len(s.Enum))
		for i, v := range s.Enum {
			values[i] = code(jsonValue(v))
		}
		details = append(details, strings.Join(values, ", "))
	}
	if len(s.Const) > 0 {
		details = append(details, code(jsonValue(s.Const)))
	}
	details = append(details, s.ranges()...)
	if s.Pattern != "" {
		details = append(details, "pattern "+code(s.Pattern))
	}
	if len(s.Default) > 0 {
		details = append(details, "default "+code(jsonValue(s.Default)))
	}
	if len(details) == 0 {
		return s.baseType()
	}
	return s.baseType() + " (" + strings.Join(details, ", ") + ")"
}

// ranges describes minimum/maximum, string length and item count.
func (s schema) ranges() []string {
	var out []string
	lo, hi := s.Minimum, s.Maximum
	loText, hiText := "≥ ", "≤ "
	if v := number(s.ExclusiveMinimum); v != nil {
		lo, loText = v, "> "
	}
	if v := number(s.ExclusiveMaximum); v != nil {
		hi, hiText = v, "< "
	}
	if r := span(lo, hi, loText, hiText, loText == "≥ " && hiText == "≤ "); r != "" {
		out = append(out, r)
	}
	if r := span(intPtr(s.MinLength), intPtr(s.MaxLength), "≥ ", "≤ ", true); r != "" {
		out = append(out, "length "+r)
	}
	if r := span(intPtr(s.MinItems), intPtr(s.MaxItems), "≥ ", "≤ ", true); r != "" {
		out = append(out, "items "+r)
	}
	return out
}

// span writes "1–200", "≥ 1" or "≤ 200"; a closed range only when both ends
// are inclusive.
func span(lo, hi *float64, loText, hiText string, inclusive bool) string {
	switch {
	case lo != nil && hi != nil && inclusive:
		return formatNumber(*lo) + "–" + formatNumber(*hi)
	case lo != nil && hi != nil:
		return loText + formatNumber(*lo) + ", " + hiText + formatNumber(*hi)
	case lo != nil:
		return loText + formatNumber(*lo)
	case hi != nil:
		return hiText + formatNumber(*hi)
	}
	return ""
}

func number(raw json.RawMessage) *float64 {
	var v float64
	if json.Unmarshal(raw, &v) != nil {
		return nil
	}
	return &v
}

func intPtr(v *int) *float64 {
	if v == nil {
		return nil
	}
	f := float64(*v)
	return &f
}

func formatNumber(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

// jsonValue shows a JSON value: strings without quotes, the rest as sent.
func jsonValue(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var compact bytes.Buffer
	if json.Compact(&compact, raw) == nil {
		return compact.String()
	}
	return string(raw)
}

// UndescribedParams names the input parameters of a tool's inputSchema that
// have no description, as "tool.param". The model can only guess what they
// are for.
func UndescribedParams(toolName string, inputSchema any) []string {
	raw, err := json.Marshal(inputSchema)
	if err != nil {
		return nil
	}
	return undescribed(toolName, raw)
}

func undescribed(toolName string, raw json.RawMessage) []string {
	s, err := parseSchema(raw)
	if err != nil {
		return nil
	}
	props, _ := s.properties()
	var out []string
	for _, p := range props {
		if strings.TrimSpace(p.Schema.Description) == "" {
			out = append(out, toolName+"."+p.Name)
		}
	}
	return out
}

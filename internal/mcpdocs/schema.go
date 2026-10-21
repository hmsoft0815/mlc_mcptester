package mcpdocs

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// schema is the part of a JSON Schema the reference shows. Nested schemas
// stay raw, so their properties can be read in source order.
type schema struct {
	Type             json.RawMessage
	Format           string
	Enum             []json.RawMessage
	Const            json.RawMessage
	Pattern          string
	Minimum          *float64
	Maximum          *float64
	ExclusiveMinimum json.RawMessage // a number, or a bool before draft 6
	ExclusiveMaximum json.RawMessage
	MinLength        *float64
	MaxLength        *float64
	MinItems         *float64
	MaxItems         *float64
	Default          json.RawMessage
	Description      string
	Items            json.RawMessage
	Properties       json.RawMessage
	Required         []string
	AnyOf            []json.RawMessage
	OneOf            []json.RawMessage
	AllOf            []json.RawMessage
	Ref              string
}

// property is one entry of a schema's properties, in source order.
type property struct {
	Name     string
	Schema   schema
	Required bool
}

// parseSchema reads a schema leniently: the reference describes what a
// server sends, so a boolean schema, a draft-3 "required": true or a
// minLength of 1.0 must not stop it. A field of unexpected shape is left out.
func parseSchema(raw json.RawMessage) schema {
	var s schema
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return s // true, false, null or not an object: any value
	}
	get := func(key string, dst any) { _ = json.Unmarshal(m[key], dst) }
	s.Type, s.Const, s.Default = m["type"], m["const"], m["default"]
	s.ExclusiveMinimum, s.ExclusiveMaximum = m["exclusiveMinimum"], m["exclusiveMaximum"]
	s.Items, s.Properties = m["items"], m["properties"]
	get("format", &s.Format)
	get("enum", &s.Enum)
	get("pattern", &s.Pattern)
	get("description", &s.Description)
	get("required", &s.Required)
	get("anyOf", &s.AnyOf)
	get("oneOf", &s.OneOf)
	get("allOf", &s.AllOf)
	get("$ref", &s.Ref)
	s.Minimum, s.Maximum = number(m["minimum"]), number(m["maximum"])
	s.MinLength, s.MaxLength = number(m["minLength"]), number(m["maxLength"])
	s.MinItems, s.MaxItems = number(m["minItems"]), number(m["maxItems"])
	return s
}

// properties returns the schema's properties in the order the server sent
// them; decoding into a map would sort them.
func (s schema) properties() ([]property, error) {
	if len(s.Properties) == 0 || string(s.Properties) == "null" {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(s.Properties))
	tok, err := dec.Token()
	if err != nil {
		return nil, fmt.Errorf("reading properties: %w", err)
	}
	if tok != json.Delim('{') {
		return nil, nil // not an object: nothing to list
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
		props = append(props, property{Name: name, Schema: parseSchema(raw), Required: slices.Contains(s.Required, name)})
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
			if items := parseSchema(s.Items).baseType(); items != "any" {
				t = "array of " + items
			}
		}
		parts = append(parts, t)
	}
	for _, alt := range append(slices.Clone(s.AnyOf), s.OneOf...) {
		parts = append(parts, parseSchema(alt).baseType())
	}
	var all []string
	for _, sub := range s.AllOf {
		all = append(all, parseSchema(sub).baseType())
	}
	switch {
	case len(parts) > 0:
		return strings.Join(parts, " | ")
	case len(all) > 0:
		return strings.Join(all, " & ")
	case s.Ref != "":
		return cmp.Or(s.Ref[strings.LastIndex(s.Ref, "/")+1:], s.Ref)
	case len(s.Const) > 0:
		return "constant"
	case len(s.Properties) > 0:
		return "object"
	}
	return "any"
}

// typeText is the type with its constraints, as shown in the Type column;
// it is ready for a table cell.
func (s schema) typeText() string {
	var details []string
	if s.Format != "" {
		details = append(details, cell(s.Format))
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
		return cell(s.baseType())
	}
	return cell(s.baseType()) + " (" + strings.Join(details, ", ") + ")"
}

// ranges describes minimum/maximum, string length and item count.
func (s schema) ranges() []string {
	var out []string
	lo, hi := s.Minimum, s.Maximum
	loText, hiText := "≥ ", "≤ "
	if v, exclusive := exclusiveBound(s.ExclusiveMinimum, lo); exclusive {
		lo, loText = v, "> "
	}
	if v, exclusive := exclusiveBound(s.ExclusiveMaximum, hi); exclusive {
		hi, hiText = v, "< "
	}
	if r := span(lo, hi, loText, hiText, loText == "≥ " && hiText == "≤ "); r != "" {
		out = append(out, r)
	}
	if r := span(s.MinLength, s.MaxLength, "≥ ", "≤ ", true); r != "" {
		out = append(out, "length "+r)
	}
	if r := span(s.MinItems, s.MaxItems, "≥ ", "≤ ", true); r != "" {
		out = append(out, "items "+r)
	}
	return out
}

// exclusiveBound reads exclusiveMinimum/-Maximum: a number since draft 6,
// before that true makes minimum/maximum itself exclusive.
func exclusiveBound(raw json.RawMessage, inclusive *float64) (*float64, bool) {
	if v := number(raw); v != nil {
		return v, true
	}
	var flag bool
	if json.Unmarshal(raw, &flag) == nil && flag && inclusive != nil {
		return inclusive, true
	}
	return nil, false
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
	props, _ := parseSchema(raw).properties()
	var out []string
	for _, p := range props {
		if strings.TrimSpace(p.Schema.Description) == "" {
			out = append(out, toolName+"."+p.Name)
		}
	}
	return out
}

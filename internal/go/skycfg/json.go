package skycfg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"go.starlark.net/starlark"
	yaml "gopkg.in/yaml.v2"
)

// JsonModule returns a Starlark module for JSON helpers.
func JsonModule() starlark.Value {
	return &Module{
		Name: "json",
		Attrs: starlark.StringDict{
			"marshal": jsonMarshal(),
		},
	}
}

// jsonMarshal returns a Starlark function for marshaling plain values
// (dicts, lists, etc) to JSON.
//
//  def json.marshal(value) -> str
func jsonMarshal() starlark.Callable {
	return starlark.NewBuiltin("json.marshal", fnJsonMarshal)
}

func fnJsonMarshal(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var v starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "value", &v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeJSON(&buf, v); err != nil {
		return nil, err
	}
	return starlark.String(buf.String()), nil
}

// YamlModule returns a Starlark module for YAML helpers.
func YamlModule() starlark.Value {
	return &Module{
		Name: "yaml",
		Attrs: starlark.StringDict{
			"marshal": yamlMarshal(),
		},
	}
}

// yamlMarshal returns a Starlark function for marshaling plain values
// (dicts, lists, etc) to YAML.
//
//  def yaml.marshal(value) -> str
func yamlMarshal() starlark.Callable {
	return starlark.NewBuiltin("yaml.marshal", fnYamlMarshal)
}

func fnYamlMarshal(t *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var v starlark.Value
	if err := starlark.UnpackArgs(fn.Name(), args, kwargs, "value", &v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeJSON(&buf, v); err != nil {
		return nil, err
	}
	var jsonObj interface{}
	if err := yaml.Unmarshal(buf.Bytes(), &jsonObj); err != nil {
		return nil, err
	}
	yamlBytes, err := yaml.Marshal(jsonObj)
	if err != nil {
		return nil, err
	}
	return starlark.String(yamlBytes), nil
}

// Adapted from struct-specific JSON function:
// https://github.com/google/starlark/blob/67717b5898061eb621519a94a4b89cedede9bca0/starlarkstruct/struct.go#L321
func writeJSON(out *bytes.Buffer, v starlark.Value) error {
	if marshaler, ok := v.(json.Marshaler); ok {
		jsonData, err := marshaler.MarshalJSON()
		if err != nil {
			return err
		}
		out.Write(jsonData)
		return nil
	}

	switch v := v.(type) {
	case starlark.NoneType:
		out.WriteString("null")
	case starlark.Bool:
		fmt.Fprintf(out, "%t", v)
	case starlark.Int:
		out.WriteString(v.String())
	case starlark.Float:
		fmt.Fprintf(out, "%g", v)
	case starlark.String:
		s := string(v)
		if goQuoteIsSafe(s) {
			fmt.Fprintf(out, "%q", s)
		} else {
			// vanishingly rare for text strings
			data, _ := json.Marshal(s)
			out.Write(data)
		}
	case starlark.Indexable: // Tuple, List
		out.WriteByte('[')
		for i, n := 0, starlark.Len(v); i < n; i++ {
			if i > 0 {
				out.WriteString(", ")
			}
			if err := writeJSON(out, v.Index(i)); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case *starlark.Dict:
		out.WriteByte('{')
		for i, itemPair := range v.Items() {
			key := itemPair[0]
			value := itemPair[1]
			if i > 0 {
				out.WriteString(", ")
			}
			if err := writeJSON(out, key); err != nil {
				return err
			}
			out.WriteString(": ")
			if err := writeJSON(out, value); err != nil {
				return err
			}
		}
		out.WriteByte('}')
	default:
		return fmt.Errorf("TypeError: value %s (type `%s') can't be converted to JSON.", v.String(), v.Type())
	}
	return nil
}

func goQuoteIsSafe(s string) bool {
	for _, r := range s {
		// JSON doesn't like Go's \xHH escapes for ASCII control codes,
		// nor its \UHHHHHHHH escapes for runes >16 bits.
		if r < 0x20 || r >= 0x10000 {
			return false
		}
	}
	return true
}

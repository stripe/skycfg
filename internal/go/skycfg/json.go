package skycfg

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/google/skylark"
	yaml "gopkg.in/yaml.v2"
)

// JsonModule returns a Skylark module for JSON helpers.
func JsonModule() skylark.Value {
	return &Module{
		Name: "json",
		Attrs: skylark.StringDict{
			"marshal": jsonMarshal(),
		},
	}
}

// jsonMarshal returns a Skylark function for marshaling plain values
// (dicts, lists, etc) to JSON.
//
//  def json.marshal(value) -> str
func jsonMarshal() skylark.Callable {
	return skylark.NewBuiltin("json.marshal", fnJsonMarshal)
}

func fnJsonMarshal(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var v skylark.Value
	if err := skylark.UnpackArgs(fn.Name(), args, kwargs, "value", &v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeJSON(&buf, v); err != nil {
		return nil, err
	}
	return skylark.String(buf.String()), nil
}

// YamlModule returns a Skylark module for YAML helpers.
func YamlModule() skylark.Value {
	return &Module{
		Name: "yaml",
		Attrs: skylark.StringDict{
			"marshal": yamlMarshal(),
		},
	}
}

// yamlMarshal returns a Skylark function for marshaling plain values
// (dicts, lists, etc) to YAML.
//
//  def yaml.marshal(value) -> str
func yamlMarshal() skylark.Callable {
	return skylark.NewBuiltin("yaml.marshal", fnYamlMarshal)
}

func fnYamlMarshal(t *skylark.Thread, fn *skylark.Builtin, args skylark.Tuple, kwargs []skylark.Tuple) (skylark.Value, error) {
	var v skylark.Value
	if err := skylark.UnpackArgs(fn.Name(), args, kwargs, "value", &v); err != nil {
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
	return skylark.String(yamlBytes), nil
}

// Adapted from struct-specific JSON function:
// https://github.com/google/skylark/blob/67717b5898061eb621519a94a4b89cedede9bca0/skylarkstruct/struct.go#L321
func writeJSON(out *bytes.Buffer, v skylark.Value) error {
	switch v := v.(type) {
	case skylark.NoneType:
		out.WriteString("null")
	case skylark.Bool:
		fmt.Fprintf(out, "%t", v)
	case skylark.Int:
		out.WriteString(v.String())
	case skylark.Float:
		fmt.Fprintf(out, "%g", v)
	case skylark.String:
		s := string(v)
		if goQuoteIsSafe(s) {
			fmt.Fprintf(out, "%q", s)
		} else {
			// vanishingly rare for text strings
			data, _ := json.Marshal(s)
			out.Write(data)
		}
	case skylark.Indexable: // Tuple, List
		out.WriteByte('[')
		for i, n := 0, skylark.Len(v); i < n; i++ {
			if i > 0 {
				out.WriteString(", ")
			}
			if err := writeJSON(out, v.Index(i)); err != nil {
				return err
			}
		}
		out.WriteByte(']')
	case *skylark.Dict:
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
		return fmt.Errorf("cannot convert %s to JSON", v.Type())
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

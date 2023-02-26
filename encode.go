package gokonfi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAML encoding.

func EncodeAsYaml(v Val) (string, error) {
	bs, err := yaml.Marshal(v)
	return string(bs), err
}

func (r *RecVal) MarshalYAML() (interface{}, error) {
	return r.Fields, nil
}

func (xs ListVal) MarshalYAML() (interface{}, error) {
	return xs.Elements, nil
}

func (x UnitVal) MarshalYAML() (interface{}, error) {
	return x.V, nil
}

func (f *FuncExprVal) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("Cannot encode function expressions in YAML")
}

func (f *NativeFuncVal) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("Cannot encode native functions in YAML")
}

// JSON encoding.

func (r *RecVal) MarshalJSON() ([]byte, error) {
	// json.Marshal will always HTML-encode < > &, so we use this "workaround" :(
	// Creating a new encoder for each (nested) record is probably not very fast.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r.Fields); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (xs ListVal) MarshalJSON() ([]byte, error) {
	// json.Marshal will always HTML-encode < > &, so we use this "workaround" :(
	// Creating a new encoder for each (nested) record is probably not very fast.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(xs.Elements); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (t UnitVal) MarshalJSON() ([]byte, error) {
	// json.Marshal will always HTML-encode < > &, so we use this "workaround" :(
	// Creating a new encoder for each (nested) record is probably not very fast.
	return json.Marshal(t.V)
}

func (r NilVal) MarshalJSON() ([]byte, error) {
	return json.Marshal(nil)
}

// EncodeAsJson encodes the given Val as a compact JSON value (without newlines).
func EncodeAsJson(v Val) (string, error) {
	return encodeAsJsonIndent(v, "", "")
}

// EncodeAsJsonIndent encodes the given Val as an indented, multi-line JSON value.
func EncodeAsJsonIndent(v Val) (string, error) {
	return encodeAsJsonIndent(v, "", "  ")
}

func encodeAsJsonIndent(v Val, prefix, indent string) (string, error) {
	var sb strings.Builder
	enc := json.NewEncoder(&sb)
	enc.SetEscapeHTML(false)
	doIndent := len(prefix) > 0 || len(indent) > 0
	if doIndent {
		enc.SetIndent(prefix, indent)
	}
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	s := sb.String()
	return strings.TrimRight(s, "\n"), nil
}

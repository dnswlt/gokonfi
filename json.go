package gokonfi

import (
	"bytes"
	"encoding/json"
	"strings"
)

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

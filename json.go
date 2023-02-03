package gokonfi

import (
	"encoding/json"
)

func (r *RecVal) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.Fields)
}

func (r NilVal) MarshalJSON() ([]byte, error) {
	return json.Marshal(nil)
}

// EncodeAsJson encodes the given Val as a compact JSON value (without newlines).
func EncodeAsJson(v Val) (string, error) {
	bs, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

// EncodeAsJsonIndent encodes the given Val as an indented, multi-line JSON value.
func EncodeAsJsonIndent(v Val) (string, error) {
	bs, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

package util

import (
	"reflect"
	"testing"
)

func TestMarshalWithIndent(t *testing.T) {
	tests := map[string]struct {
		data     interface{}
		indent   string
		expected []byte
	}{
		"no indent": {
			data:     struct{ Name string }{"John"},
			expected: []byte(`{"Name":"John"}`),
		},
		"indent 1": {
			data:   struct{ Name string }{"John"},
			indent: "1",
			expected: []byte(`{
 "Name": "John"
}`),
		},
		"indent negative": {
			data:     struct{ Name string }{"John"},
			indent:   "-1",
			expected: []byte(`{"Name":"John"}`),
		},
		"indent greater than 10": {
			data:     struct{ Name string }{"John"},
			indent:   "11",
			expected: []byte(`{"Name":"John"}`),
		},
		"indent not a number": {
			data:     struct{ Name string }{"John"},
			indent:   "a",
			expected: []byte(`{"Name":"John"}`),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			b, err := MarshalWithIndent(test.data, test.indent)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(test.expected, b) {
				t.Fatalf("expected %v, got %s", test.expected, b)
			}
		})
	}
}

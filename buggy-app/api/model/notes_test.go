package model

import (
	"reflect"
	"testing"
)

func TestTags(t *testing.T) {

	tests := map[string]struct {
		text     string
		expected []string
	}{
		"no tags": {
			text:     "This is an example",
			expected: []string{},
		},
		"one tag": {
			text:     "This is an example #tag1",
			expected: []string{"tag1"},
		},
		"two tags": {
			text:     "This is an example #tag1 #tag2",
			expected: []string{"tag1", "tag2"},
		},
		"two tags with spaces": {
			text:     "This is an example #tag1    #tag2    ",
			expected: []string{"tag1", "tag2"},
		},
		"tag at the start": {
			text:     "#tag1 This is an example",
			expected: []string{"tag1"},
		},
		"tag with comma": {
			text:     "This is an example #tag1,#tag2",
			expected: []string{"tag1", "tag2"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tags := extractTags(test.text)

			if !reflect.DeepEqual(test.expected, tags) {
				t.Fatalf("expected %v, got %v", test.expected, tags)
			}
		})
	}
}



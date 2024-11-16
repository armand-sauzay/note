package main

import "testing"

func TestExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "extract title from markdown content",
			content:  "# My Title\nSome content",
			expected: "My Title",
		},
		{
			name:     "no title returns empty string",
			content:  "Some content without title",
			expected: "",
		},
		{
			name:     "empty content returns empty string",
			content:  "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTitle(tt.content)
			if got != tt.expected {
				t.Errorf("extractTitle() = %v, want %v", got, tt.expected)
			}
		})
	}
}

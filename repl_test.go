package main

import (
	"testing"
)

func TestCleanInput(t *testing.T) {

	t.Skip()

	cases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "  hello  world  ",
			expected: []string{"hello", "world"},
		},
		{
			input:    "pikachu charmander bulbasaur ",
			expected: []string{"pikachu", "charmander", "bulbasaur"},
		},
		{
			input:    "Are you   there",
			expected: []string{"Are", "you", "there"},
		},
	}

	for _, c := range cases {
		actual := cleanInput(c.input)
		if len(actual) != len(c.expected) {
			t.Errorf("Didn't matched")
		}
		for i := range actual {
			word := actual[i]
			expectedWord := c.expected[i]
			if word != expectedWord {
				t.Errorf("Didn't matched")
			}
		}
	}
}

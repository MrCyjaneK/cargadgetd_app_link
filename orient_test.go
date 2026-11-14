package main

import "testing"

func TestParseOrient(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"'portrait'\n", "portrait", true},
		{"landscape", "landscape", true},
		{"  'landscape-inverted'  ", "landscape-inverted", true},
		{"dynamic", "dynamic", true},
		{"portrait-inverted", "portrait-inverted", true},
		{"sideways", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := parseOrient(c.in)
		if ok != c.ok || got != c.want {
			t.Fatalf("parseOrient(%q) = %q,%v want %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

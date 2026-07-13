//go:build windows && wailsgui

package main

import "testing"

func TestVersionLess(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{current: "v1.5.1", latest: "v1.5.2", want: true},
		{current: "1.5.2", latest: "v1.5.2", want: false},
		{current: "v1.6.0", latest: "v1.5.9", want: false},
		{current: "dev", latest: "v1.5.2", want: true},
	}

	for _, test := range tests {
		if got := versionLess(test.current, test.latest); got != test.want {
			t.Errorf("versionLess(%q, %q) = %t, want %t", test.current, test.latest, got, test.want)
		}
	}
}

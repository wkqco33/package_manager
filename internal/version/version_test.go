package version

import "testing"

func TestCompareSemVer(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v1.2.3", "1.2.4", -1}, {"1.10.0", "1.9.9", 1}, {"1.0.0-alpha", "1.0.0", -1}, {"1.0.0", "v1.0.0", 0},
	}
	for _, tc := range cases {
		if got := Compare(tc.a, tc.b); got != tc.want {
			t.Errorf("Compare(%q,%q)=%d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestSatisfies(t *testing.T) {
	cases := []struct {
		v, constraint string
		want          bool
	}{
		{"1.4.2", "^1.4.0", true}, {"2.0.0", "^1.4.0", false}, {"1.4.9", "~1.4.0", true},
		{"1.5.0", "~1.4.0", false}, {"1.8.0", ">=1.0.0 <2.0.0", true}, {"2.0.0", ">=1.0.0 <2.0.0", false},
	}
	for _, tc := range cases {
		if got := Satisfies(tc.v, tc.constraint); got != tc.want {
			t.Errorf("Satisfies(%q,%q)=%v, want %v", tc.v, tc.constraint, got, tc.want)
		}
	}
}

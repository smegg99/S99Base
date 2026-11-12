// release/version/version_test.go

package version

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.4.3", "1.4.3", 0},
		{"v1.4.3", "1.4.3", 0},
		{"1.4.3", "1.5.0", -1},
		{"1.10.0", "1.9.9", 1},
		{"1.5", "1.5.0", 0},
		{"1.5.0-rc.1", "1.5.0", -1},
		{"1.5.0-rc.2", "1.5.0-rc.1", 1},
		{"", "1.0.0", 0},
		{"main", "1.0.0", 0},
		{"1", "1.0.0", 0},
		{"  v1.5.0\n", "1.5.0", 0},
		{"1.5.0+build.1", "1.5.0", -1},
		{"1.5.0-rc.10", "1.5.0-rc.2", -1},
		{"1.5.0-", "1.5.0", 0},
		{"01.05.00", "1.5.0", 0},
		{"1.5.0.1", "1.5.0", 0},
		{"1.5.0", "main", 0},
		{"-1.0.0", "1.0.0", 0},
		{"9007199254740993", "9007199254740992", 0},
	}
	for _, tc := range cases {
		t.Run(tc.a+"/"+tc.b, func(t *testing.T) {
			got := Compare(tc.a, tc.b)
			if got < 0 {
				got = -1
			} else if got > 0 {
				got = 1
			}
			if got != tc.want {
				t.Errorf("Compare(%q, %q) sign = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestTrim(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"v1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"vv1", "v1"},
		{"", ""},
		{" v1", " v1"},
	} {
		if got := Trim(tc.in); got != tc.want {
			t.Errorf("Trim(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

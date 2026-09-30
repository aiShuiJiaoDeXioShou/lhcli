package selfupdate

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		left  string
		right string
		want  int
	}{
		{"v0.1.0", "v0.2.0", -1},
		{"v0.2.0", "v0.2.0", 0},
		{"v1.0.0", "v0.9.9", 1},
		{"v0.2.0-rc.1", "v0.2.0", -1},
		{"v0.2.0", "v0.2.0-rc.1", 1},
		{"v0.2.0-rc.2", "v0.2.0-rc.1", 1},
		{"v0.2.0-rc.1", "v0.2.0-beta.1", 1},
		{"v0.2.0-alpha", "v0.2.0-alpha.1", -1},
		{"v0.2.0+meta", "v0.2.0", 0},
		{"1.2.3", "v1.2.4", -1},
	}

	for _, tc := range cases {
		got, err := Compare(tc.left, tc.right)
		if err != nil {
			t.Fatalf("Compare(%q, %q) 返回错误: %v", tc.left, tc.right, err)
		}
		if got != tc.want {
			t.Errorf("Compare(%q, %q) = %d，期望 %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestCompareInvalid(t *testing.T) {
	for _, value := range []string{"", "dev", "v1.x.0", "1.2.3.4", "v-1.0.0"} {
		if _, err := Compare(value, "v1.0.0"); err == nil {
			t.Errorf("Compare(%q, ...) 期望报错，实际通过", value)
		}
		if IsValid(value) {
			t.Errorf("IsValid(%q) 期望为 false", value)
		}
	}
}

func TestIsValid(t *testing.T) {
	for _, value := range []string{"v0.1.0", "1.2.3", "v1.2.3-rc.1", "v1.2.3+build"} {
		if !IsValid(value) {
			t.Errorf("IsValid(%q) 期望为 true", value)
		}
	}
}

package usercanon

import (
	"os"
	"path/filepath"
	"testing"
)

func withUsersDir(t *testing.T, setup func(dir string)) func() {
	t.Helper()
	tmp, err := os.MkdirTemp("", "usercanon-")
	if err != nil {
		t.Fatal(err)
	}
	setup(tmp)
	orig := UsersDir
	UsersDir = tmp
	return func() {
		UsersDir = orig
		_ = os.RemoveAll(tmp)
	}
}

func TestResolve(t *testing.T) {
	cleanup := withUsersDir(t, func(dir string) {
		for _, name := range []string{"gray", "alex", "leo"} {
			_ = os.MkdirAll(filepath.Join(dir, name), 0o755)
		}
		_ = os.WriteFile(
			filepath.Join(dir, "gray", "metadata.json"),
			[]byte(`{"telegram_id":"595103437","telegram_username":"graythedev"}`),
			0o644,
		)
		_ = os.WriteFile(
			filepath.Join(dir, "leo", "metadata.json"),
			[]byte(`{"telegram_id":123456}`),
			0o644,
		)
	})
	defer cleanup()

	cases := []struct{ in, want string }{
		{"gray", "gray"},
		{"Gray", "gray"},
		{"i am gray", "gray"},
		{"I am Gray", "gray"},
		{"i am gray (595103437)", "gray"},
		{"fake name (123456)", "leo"},
		{"Alex says hi", "alex"},
		{"unknown user (999)", "unknown_user_999"},
		{"", "unknown"},
	}
	for _, c := range cases {
		if got := Resolve(c.in); got != c.want {
			t.Errorf("Resolve(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Gray", "gray"},
		{"i am gray", "i_am_gray"},
		{"i am gray (595)", "i_am_gray_595"},
		{"", "unknown"},
		{"   ", "unknown"},
	}
	for _, c := range cases {
		if got := Slugify(c.in); got != c.want {
			t.Errorf("Slugify(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

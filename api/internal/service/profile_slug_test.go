package service

import "testing"

func TestSlugFromDisplayName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"Ryan Draga", "ryan-draga"},
		{"  Alice  ", "alice"},
		{"Admin", "admin"},
		{"ab", "user"},
		{"", "user"},
		{"Hello_World!!!", "hello-world"},
		{"a--b", "a-b"},
	}
	for _, tc := range cases {
		if got := SlugFromDisplayName(tc.in); got != tc.want {
			t.Errorf("SlugFromDisplayName(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeProfileSlug(t *testing.T) {
	valid32 := "a" + stringsRepeat("b", 30) + "z"
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "", false},
		{"  Alice  ", "alice", false},
		{"ab", "", true},
		{"a", "", true},
		{"abc", "abc", false},
		{"a-b-c", "a-b-c", false},
		{"-abc", "", true},
		{"abc-", "", true},
		{"ab_c", "", true},
		{"Admin", "", true},
		{"settings", "", true},
		{"request-access", "", true},
		{stringsRepeat("x", 33), "", true},
		{valid32, valid32, false},
	}

	for _, tc := range cases {
		got, err := NormalizeProfileSlug(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("NormalizeProfileSlug(%q): expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeProfileSlug(%q): unexpected error %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("NormalizeProfileSlug(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func stringsRepeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}

func TestObjectKeyFromUserAssetURL(t *testing.T) {
	base := "https://cdn.example.com"
	if got := objectKeyFromUserAssetURL(base, base+"/banners/u1.jpg", "banners/"); got != "banners/u1.jpg" {
		t.Fatalf("banner key: got %q", got)
	}
	if objectKeyFromUserAssetURL(base, base+"/avatars/u1.jpg", "banners/") != "" {
		t.Fatal("expected empty when prefix mismatches")
	}
}

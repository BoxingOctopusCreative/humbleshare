package media

import "testing"

func TestRefererAllowed(t *testing.T) {
	hosts := AllowedHosts("https://humbleshare.boxingoctopus.io", "http://localhost:5173")
	cases := []struct {
		ref, origin string
		want        bool
	}{
		{"", "", true},
		{"https://humbleshare.boxingoctopus.io/settings", "", true},
		{"", "http://localhost:5173", true},
		{"https://evil.example/page", "", false},
		{"https://evil.example/page", "https://humbleshare.boxingoctopus.io", false},
	}
	for _, tc := range cases {
		if got := RefererAllowed(tc.ref, tc.origin, hosts); got != tc.want {
			t.Errorf("RefererAllowed(%q,%q)=%v want %v", tc.ref, tc.origin, got, tc.want)
		}
	}
}

func TestParseObjectKey(t *testing.T) {
	key, ok := ParseObjectKey("avatars", "u1.png")
	if !ok || key != "avatars/u1.png" {
		t.Fatalf("got %q %v", key, ok)
	}
	if _, ok := ParseObjectKey("nope", "u1.png"); ok {
		t.Fatal("expected reject")
	}
	if _, ok := ParseObjectKey("avatars", "../x.png"); ok {
		t.Fatal("expected reject traversal")
	}
}

func TestProxyURL(t *testing.T) {
	got := ProxyURL("https://app.example", "https://cdn.example", "https://cdn.example/avatars/u1.png")
	want := "https://app.example/media/avatars/u1.png"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

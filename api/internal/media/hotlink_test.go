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
	key, ok = ParseObjectKey("svg", "humbleshare_wide.svg")
	if !ok || key != "SVG/humbleshare_wide.svg" {
		t.Fatalf("svg key: got %q %v", key, ok)
	}
	if _, ok := ParseObjectKey("svg", "note.txt"); ok {
		t.Fatal("expected reject non-svg")
	}
}

func TestProxyURL(t *testing.T) {
	got := ProxyURL("https://cdn.example", "https://cdn.example/avatars/u1.png")
	want := "/media/avatars/u1.png"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if ProxyURL("https://cdn.example", "https://other.example/avatars/u1.png") != "/media/avatars/u1.png" {
		t.Fatal("expected avatars path to rewrite regardless of CDN host")
	}
	passthrough := "https://cdn.example/other/u1.png"
	if ProxyURL("https://cdn.example", passthrough) != passthrough {
		t.Fatal("expected non-avatar path to pass through")
	}
}

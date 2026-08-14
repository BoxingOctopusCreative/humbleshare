package service

import "testing"

func TestSniffImage(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		ct   string
		ext  string
		ok   bool
	}{
		{"jpeg", []byte{0xff, 0xd8, 0xff, 0x00}, "image/jpeg", "jpg", true},
		{"png", []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}, "image/png", "png", true},
		{"webp", []byte("RIFF....WEBP"), "image/webp", "webp", true},
		{"empty", nil, "", "", false},
		{"text", []byte("hello"), "", "", false},
	}
	for _, tc := range cases {
		ct, ext, ok := sniffImage(tc.data)
		if ok != tc.ok || ct != tc.ct || ext != tc.ext {
			t.Errorf("%s: got (%q,%q,%v) want (%q,%q,%v)", tc.name, ct, ext, ok, tc.ct, tc.ext, tc.ok)
		}
	}
}

func TestObjectKeyFromAvatarURL(t *testing.T) {
	base := "https://cdn.example.com"
	got := objectKeyFromAvatarURL(base, base+"/avatars/u1.jpg")
	if got != "avatars/u1.jpg" {
		t.Fatalf("got %q", got)
	}
	if objectKeyFromAvatarURL(base, "https://old.example/avatars/u1.jpg") != "avatars/u1.jpg" {
		t.Fatal("expected key from path even when CDN host changed")
	}
	if objectKeyFromAvatarURL(base, base+"/../etc/passwd") != "" {
		t.Fatal("expected empty for traversal")
	}
}

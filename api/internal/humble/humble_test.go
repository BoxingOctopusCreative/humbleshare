package humble

import "testing"

func TestIsSteam(t *testing.T) {
	cases := []struct {
		name string
		tpk  humbleTpk
		want bool
	}{
		{
			name: "steam human name",
			tpk:  humbleTpk{KeyTypeHumanName: "Steam"},
			want: true,
		},
		{
			name: "steam machine name",
			tpk:  humbleTpk{MachineName: "game_steam"},
			want: true,
		},
		{
			name: "epic",
			tpk:  humbleTpk{KeyTypeHumanName: "Epic Games", MachineName: "game_epic"},
			want: false,
		},
		{
			name: "empty",
			tpk:  humbleTpk{},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tpk.isSteam(); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestClassifyPaste(t *testing.T) {
	cases := []struct {
		in      string
		kind    pasteKind
		want    string
		wantErr bool
	}{
		{"AbCdEfGhIjKlMnOp", pasteOrderGamekey, "AbCdEfGhIjKlMnOp", false},
		{
			"https://www.humblebundle.com/downloads?key=AbCdEfGhIjKlMnOp",
			pasteOrderGamekey,
			"AbCdEfGhIjKlMnOp",
			false,
		},
		{
			"https://www.humblebundle.com/gift?key=AbCdEfGhIjKlMnOp",
			pasteGiftURL,
			"https://www.humblebundle.com/gift?key=AbCdEfGhIjKlMnOp",
			false,
		},
		{
			"https://www.humblebundle.com/gift?key=AbCdEfGhIjKlMnOp&foo=1",
			pasteGiftURL,
			"https://www.humblebundle.com/gift?key=AbCdEfGhIjKlMnOp",
			false,
		},
		{"AAAAA-BBBBB-CCCCC", 0, "", true},
		{"https://www.humblebundle.com/downloads", 0, "", true},
		{"", 0, "", true},
	}
	for _, tc := range cases {
		kind, got, err := classifyPaste(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("classifyPaste(%q) err=nil want error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("classifyPaste(%q) err=%v", tc.in, err)
			continue
		}
		if kind != tc.kind || got != tc.want {
			t.Errorf("classifyPaste(%q)=(%v,%q) want (%v,%q)", tc.in, kind, got, tc.kind, tc.want)
		}
	}
}

func TestIsHumbleGiftURL(t *testing.T) {
	if !isHumbleGiftURL("https://www.humblebundle.com/gift?key=abc") {
		t.Fatal("expected gift URL")
	}
	if isHumbleGiftURL("https://www.humblebundle.com/downloads?key=abc") {
		t.Fatal("downloads is not a gift URL")
	}
	if codeTypeForValue("https://www.humblebundle.com/gift?key=abc") != "gift_link" {
		t.Fatal("expected gift_link")
	}
}

func TestNormalizeSessionCookie(t *testing.T) {
	got := normalizeSessionCookie(`_simpleauth_sess=%22abc%22; Path=/`)
	if got != `"abc"` {
		t.Fatalf("got %q", got)
	}
	if normalizeSessionCookie(`"abc"`) != `"abc"` {
		t.Fatal("bare value should pass through")
	}
}

func TestCleanGiftTitle(t *testing.T) {
	if got := cleanGiftTitle("Hades | Humble Bundle"); got != "Hades" {
		t.Fatalf("got %q", got)
	}
	if !isGenericGiftTitle("Your Humble Bundle Receipt") {
		t.Fatal("expected generic")
	}
	if !isGenericGiftTitle("A Very Special Gift For You") {
		t.Fatal("expected special gift placeholder to be generic")
	}
	if isGenericGiftTitle("Hades") {
		t.Fatal("real title should not be generic")
	}
}

func TestLookupTitle(t *testing.T) {
	titles := map[string]string{
		"https://www.humblebundle.com/gift?key=AbCd": "Hades",
		"raw-paste": "Celeste",
	}
	if got := lookupTitle(titles, "raw-paste", "https://www.humblebundle.com/gift?key=AbCd"); got != "Celeste" {
		t.Fatalf("prefer first key, got %q", got)
	}
	if got := lookupTitle(titles, "other", "https://www.humblebundle.com/gift?key=AbCd"); got != "Hades" {
		t.Fatalf("got %q", got)
	}
	if got := lookupTitle(nil, "x"); got != "" {
		t.Fatalf("got %q", got)
	}
}

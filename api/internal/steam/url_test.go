package steam

import "testing"

func TestParseStoreURL(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{in: "https://store.steampowered.com/app/1145360/Hades/", want: 1145360},
		{in: "https://store.steampowered.com/app/1145360", want: 1145360},
		{in: "http://store.steampowered.com/app/570/Dota_2/?utm_source=x", want: 570},
		{in: "store.steampowered.com/app/730/CounterStrike_2/", want: 730},
		{in: "https://store.steampowered.com/agecheck/app/1145360/", want: 1145360},
		{in: "https://store.steampowered.com/app/1145360/Hades", want: 1145360},
		{in: "", wantErr: true},
		{in: "https://example.com/app/1145360", wantErr: true},
		{in: "https://store.steampowered.com/genre/Action/", wantErr: true},
		{in: "https://steamcommunity.com/app/1145360", wantErr: true},
	}
	for _, tc := range cases {
		got, err := ParseStoreURL(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("ParseStoreURL(%q) err=nil want error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseStoreURL(%q) err=%v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseStoreURL(%q)=%d want %d", tc.in, got, tc.want)
		}
	}
}

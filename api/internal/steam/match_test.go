package steam

import "testing"

func TestMatchExactAppID(t *testing.T) {
	candidates := []SearchResult{
		{AppID: 570, Name: "Dota 2"},
		{AppID: 730, Name: "Counter-Strike 2"},
	}

	cases := []struct {
		name  string
		title string
		want  int
	}{
		{name: "exact", title: "Dota 2", want: 570},
		{name: "case insensitive", title: "dota 2", want: 570},
		{name: "trimmed", title: "  Counter-Strike 2  ", want: 730},
		{name: "no match", title: "Half-Life 3", want: 0},
		{name: "partial no match", title: "Dota", want: 0},
		{name: "empty", title: "   ", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MatchExactAppID(tc.title, candidates); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestLimitSuggestions(t *testing.T) {
	in := []SearchResult{
		{AppID: 1, Name: "A"},
		{AppID: 2, Name: "B"},
		{AppID: 3, Name: "C"},
	}
	got := limitSuggestions(in, 2)
	if len(got) != 2 || got[0].AppID != 1 || got[1].AppID != 2 {
		t.Fatalf("got %#v", got)
	}
	if len(limitSuggestions(in, 8)) != 3 {
		t.Fatal("short list should pass through")
	}
	if limitSuggestions(nil, 8) != nil {
		t.Fatal("nil should stay nil")
	}
}

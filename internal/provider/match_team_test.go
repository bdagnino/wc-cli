package provider

import "testing"

var sample = []Team{
	{Abbr: "BRA", Name: "Brazil"},
	{Abbr: "ARG", Name: "Argentina"},
	{Abbr: "COD", Name: "Congo DR"},
	{Abbr: "CUW", Name: "Curaçao"},
}

func TestFindTeam(t *testing.T) {
	cases := []struct {
		query string
		want  string // expected Abbr, "" means no match
	}{
		{"BRA", "BRA"},
		{"bra", "BRA"},
		{"brazil", "BRA"},
		{"arg", "ARG"},
		{"cong", "COD"},
		{"curacao", "CUW"}, // accent-insensitive
		{"zzz", ""},
	}
	for _, c := range cases {
		got, ok := FindTeam(sample, c.query)
		if c.want == "" {
			if ok {
				t.Errorf("FindTeam(%q): expected no match, got %q", c.query, got.Abbr)
			}
			continue
		}
		if !ok || got.Abbr != c.want {
			t.Errorf("FindTeam(%q) = %q (ok=%v), want %q", c.query, got.Abbr, ok, c.want)
		}
	}
}

func TestFindTeamExactBeatsPartial(t *testing.T) {
	// An exact abbreviation must rank above an incidental substring hit.
	teams := []Team{{Abbr: "USA", Name: "United States"}, {Abbr: "RSA", Name: "South Africa USA-ish"}}
	got, ok := FindTeam(teams, "USA")
	if !ok || got.Abbr != "USA" {
		t.Fatalf("expected USA to win exact match, got %q (ok=%v)", got.Abbr, ok)
	}
}

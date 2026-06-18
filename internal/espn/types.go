package espn

// Raw response shapes. Only the fields we consume are declared; ESPN returns
// much more. Keeping these minimal makes the parser resilient to unrelated
// schema churn.

type scoreboardResp struct {
	Leagues []league `json:"leagues"`
	Events  []event  `json:"events"`
}

type league struct {
	Season struct {
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
	} `json:"season"`
	Calendar []struct {
		Entries []calendarEntry `json:"entries"`
	} `json:"calendar"`
}

type calendarEntry struct {
	Label     string `json:"label"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

// round returns the stage name for this scoreboard day (e.g. "Group Stage"),
// or "" if absent.
func (r scoreboardResp) round() string {
	if len(r.Leagues) > 0 {
		return r.Leagues[0].Season.Type.Name
	}
	return ""
}

// calendarEntries returns the round windows, if present in this response.
func (r scoreboardResp) calendarEntries() []calendarEntry {
	if len(r.Leagues) > 0 && len(r.Leagues[0].Calendar) > 0 {
		return r.Leagues[0].Calendar[0].Entries
	}
	return nil
}

type event struct {
	ID           string        `json:"id"`
	Date         string        `json:"date"`
	Name         string        `json:"name"`
	Status       status        `json:"status"`
	Competitions []competition `json:"competitions"`
}

type competition struct {
	Venue struct {
		FullName string `json:"fullName"`
	} `json:"venue"`
	Competitors []competitor `json:"competitors"`
	Status      status       `json:"status"`
	Notes       []struct {
		Headline string `json:"headline"`
	} `json:"notes"`
}

type competitor struct {
	HomeAway string `json:"homeAway"`
	Score    string `json:"score"`
	Team     team   `json:"team"`
}

type team struct {
	Abbreviation string `json:"abbreviation"`
	DisplayName  string `json:"displayName"`
}

type status struct {
	DisplayClock string `json:"displayClock"`
	Type         struct {
		State       string `json:"state"` // pre | in | post
		Description string `json:"description"`
		ShortDetail string `json:"shortDetail"`
		Completed   bool   `json:"completed"`
	} `json:"type"`
}

type teamsResp struct {
	Sports []struct {
		Leagues []struct {
			Teams []struct {
				Team team `json:"team"`
			} `json:"teams"`
		} `json:"leagues"`
	} `json:"sports"`
}

type standingsResp struct {
	Children []struct {
		Name      string `json:"name"`
		Standings struct {
			Entries []standingEntry `json:"entries"`
		} `json:"standings"`
	} `json:"children"`
}

type summaryResp struct {
	KeyEvents []keyEvent `json:"keyEvents"`
	GameInfo  struct {
		Attendance int `json:"attendance"`
	} `json:"gameInfo"`
}

type keyEvent struct {
	Clock struct {
		DisplayValue string `json:"displayValue"`
	} `json:"clock"`
	Type struct {
		Text string `json:"text"`
	} `json:"type"`
	Text         string `json:"text"`
	Participants []struct {
		Athlete struct {
			DisplayName string `json:"displayName"`
		} `json:"athlete"`
	} `json:"participants"`
}

// leadersResp is the core-API season leaders feed. Each leader references its
// athlete by hypermedia link rather than embedding the name, so the name is
// resolved with a follow-up request per scorer.
type leadersResp struct {
	Categories []struct {
		Name    string        `json:"name"`
		Leaders []leaderEntry `json:"leaders"`
	} `json:"categories"`
}

// goalsLeaders returns the goals ranking, or nil if absent.
func (r leadersResp) goalsLeaders() []leaderEntry {
	for _, c := range r.Categories {
		if c.Name == "goalsLeaders" {
			return c.Leaders
		}
	}
	return nil
}

type leaderEntry struct {
	Value   float64 `json:"value"` // goal count
	Athlete struct {
		Ref string `json:"$ref"`
	} `json:"athlete"`
}

// athleteResp is the slice of an athlete document we need: the display name and
// a flag whose filename carries the 3-letter country code (".../arg.png").
type athleteResp struct {
	DisplayName string `json:"displayName"`
	Flag        struct {
		Href string `json:"href"`
	} `json:"flag"`
}

type standingEntry struct {
	Team  team `json:"team"`
	Stats []struct {
		Name  string  `json:"name"`
		Value float64 `json:"value"`
	} `json:"stats"`
}

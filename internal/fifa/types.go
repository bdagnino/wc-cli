package fifa

// Raw response shapes. Only the fields we consume are declared; FIFA returns
// far more. Keeping these minimal makes the parser resilient to schema churn.

type calendarResp struct {
	Results []calendarMatch `json:"Results"`
}

type calendarMatch struct {
	IdStage       string       `json:"IdStage"`
	IdMatch       string       `json:"IdMatch"`
	Home          calendarTeam `json:"Home"`
	Away          calendarTeam `json:"Away"`
	HomeTeamScore *int         `json:"HomeTeamScore"`
	AwayTeamScore *int         `json:"AwayTeamScore"`
}

type calendarTeam struct {
	// IdCountry is the 3-letter code (e.g. "ARG"), identical to the codes the
	// ESPN provider exposes as Team.Abbr.
	IdCountry string `json:"IdCountry"`
}

type videosResp struct {
	VodVideosBaseCarousel struct {
		Items []videoItem `json:"items"`
	} `json:"vodVideosBaseCarousel"`
}

type videoItem struct {
	VideoSubcategory string `json:"videoSubcategory"`
	// ReadMorePageURL is the site-relative watch page, e.g. "/en/watch/abc123".
	ReadMorePageURL string `json:"readMorePageUrl"`
}

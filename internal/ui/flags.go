package ui

import "strings"

// iso2 maps each ESPN 3-letter team code to an ISO 3166-1 alpha-2 country code,
// from which a flag emoji is derived. England and Scotland use subdivision tag
// sequences instead of a country pair.
var iso2 = map[string]string{
	"ALG": "DZ", "ARG": "AR", "AUS": "AU", "AUT": "AT", "BEL": "BE",
	"BIH": "BA", "BRA": "BR", "CAN": "CA", "CPV": "CV", "COL": "CO",
	"COD": "CD", "CRO": "HR", "CUW": "CW", "CZE": "CZ", "ECU": "EC",
	"EGY": "EG", "FRA": "FR", "GER": "DE", "GHA": "GH", "HAI": "HT",
	"IRN": "IR", "IRQ": "IQ", "CIV": "CI", "JPN": "JP", "JOR": "JO",
	"MEX": "MX", "MAR": "MA", "NED": "NL", "NZL": "NZ", "NOR": "NO",
	"PAN": "PA", "PAR": "PY", "POR": "PT", "QAT": "QA", "KSA": "SA",
	"SEN": "SN", "RSA": "ZA", "KOR": "KR", "ESP": "ES", "SWE": "SE",
	"SUI": "CH", "TUN": "TN", "TUR": "TR", "USA": "US", "URU": "UY",
	"UZB": "UZ",
}

// Flag returns the emoji flag for a team abbreviation, or a neutral globe when
// unknown so alignment is never broken.
func Flag(abbr string) string {
	switch abbr {
	case "ENG":
		return "🏴\U000E0067\U000E0062\U000E0065\U000E006E\U000E0067\U000E007F"
	case "SCO":
		return "🏴\U000E0067\U000E0062\U000E0073\U000E0063\U000E0074\U000E007F"
	}
	code, ok := iso2[strings.ToUpper(abbr)]
	if !ok || len(code) != 2 {
		return "🌐"
	}
	const base = 0x1F1E6 // regional indicator 'A'
	r1 := rune(base + int(code[0]-'A'))
	r2 := rune(base + int(code[1]-'A'))
	return string(r1) + string(r2)
}

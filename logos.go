package main

import "strings"

// espnCodes are ESPN's abbreviations of the teams, by their full names
// and the other names sites give them.
var espnCodes = map[string]string{
	"atlanta hawks": "atl", "boston celtics": "bos", "brooklyn nets": "bkn",
	"charlotte hornets": "cha", "chicago bulls": "chi", "cleveland cavaliers": "cle",
	"dallas mavericks": "dal", "denver nuggets": "den", "detroit pistons": "det",
	"golden state warriors": "gs", "houston rockets": "hou", "indiana pacers": "ind",
	"la clippers": "lac", "los angeles clippers": "lac", "los angeles lakers": "lal",
	"la lakers": "lal", "memphis grizzlies": "mem", "miami heat": "mia",
	"milwaukee bucks": "mil", "minnesota timberwolves": "min", "new orleans pelicans": "no",
	"new york knicks": "ny", "oklahoma city thunder": "okc", "orlando magic": "orl",
	"philadelphia 76ers": "phi", "phoenix suns": "phx", "portland trail blazers": "por",
	"sacramento kings": "sac", "san antonio spurs": "sa", "toronto raptors": "tor",
	"utah jazz": "utah", "washington wizards": "wsh",
}

// espnLogos returns ESPN's logo of a team for light and for dark
// backgrounds, by its ESPN code. The sites' own logos are drawn for one
// background: some, as the Jazz's and the Rockets', vanish on the other.
func espnLogos(code string) (light, dark string) {
	code = strings.ToLower(code)
	return "https://a.espncdn.com/i/teamlogos/nba/500/" + code + ".png",
		"https://a.espncdn.com/i/teamlogos/nba/500-dark/" + code + ".png"
}

// setLogos gives t ESPN's logos when its name or code is known, and keeps
// the logo it has otherwise.
func (t *Team) setLogos(code string) {
	if code == "" {
		code = espnCodes[strings.ToLower(t.Name)]
	}
	if code == "" {
		t.LogoDark = t.Logo
		return
	}
	t.Logo, t.LogoDark = espnLogos(code)
}

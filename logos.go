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

var teamShortNames = map[string]string{
	"atlanta hawks": "Hawks", "boston celtics": "Celtics", "brooklyn nets": "Nets",
	"charlotte hornets": "Hornets", "chicago bulls": "Bulls", "cleveland cavaliers": "Cavaliers",
	"dallas mavericks": "Mavericks", "denver nuggets": "Nuggets", "detroit pistons": "Pistons",
	"golden state warriors": "Warriors", "houston rockets": "Rockets", "indiana pacers": "Pacers",
	"la clippers": "Clippers", "los angeles clippers": "Clippers", "los angeles lakers": "Lakers",
	"la lakers": "Lakers", "memphis grizzlies": "Grizzlies", "miami heat": "Heat",
	"milwaukee bucks": "Bucks", "minnesota timberwolves": "Timberwolves", "new orleans pelicans": "Pelicans",
	"new york knicks": "Knicks", "oklahoma city thunder": "Thunder", "orlando magic": "Magic",
	"philadelphia 76ers": "76ers", "phoenix suns": "Suns", "portland trail blazers": "Trail Blazers",
	"sacramento kings": "Kings", "san antonio spurs": "Spurs", "toronto raptors": "Raptors",
	"utah jazz": "Jazz", "washington wizards": "Wizards",
}

// teamShort returns the team's standard short nickname, e.g. "Warriors".
func teamShort(name string) string {
	if s, ok := teamShortNames[strings.ToLower(strings.TrimSpace(name))]; ok {
		return s
	}
	parts := strings.Fields(name)
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return name
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

package main

import (
	"net/url"
	"os"
	"testing"

	"golang.org/x/net/html"
)

func TestParseGames(t *testing.T) {
	f, err := os.Open("testdata/nba-streams.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := html.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	final, _ := url.Parse("https://v2.streameast.ga/nba-streams/")
	games := parseGames(doc, final)
	if len(games) != 2 {
		t.Fatalf("got %d games, want 2: %+v", len(games), games)
	}
	g := games[0]
	want := Game{
		ID:   "47191",
		Path: "/nba/philadelphia-76ers-vs-new-york-knicks-4/",
		Home: Team{Name: "Philadelphia 76ers", Short: "76ers", Logo: "https://v2.streameast.ga/images/philadelphia-76ers.svg", Score: "88"},
		Away: Team{Name: "New York Knicks", Short: "Knicks", Logo: "https://v2.streameast.ga/images/new-york-knicks.svg", Score: "86"},
		Start: 1792537200, Live: true, Status: "4th",
	}
	if g != want {
		t.Errorf("first game:\n got %+v\nwant %+v", g, want)
	}
}

func TestServerOf(t *testing.T) {
	path := "/nba/philadelphia-76ers-vs-new-york-knicks-4/"
	for rawURL, want := range map[string]int{
		"https://v2.streameast.ga" + path + "2":  2,
		"https://v2.streameast.ga" + path + "3/": 3,
		"https://v2.streameast.ga" + path:        0,
		"https://v2.streameast.ga/connect.php":   0,
	} {
		if got := serverOf(path, rawURL); got != want {
			t.Errorf("serverOf(%q) = %d, want %d", rawURL, got, want)
		}
	}
}

func TestAllowedNavigation(t *testing.T) {
	site, _ := url.Parse("https://v2.streameast.ga")
	path := "/nba/philadelphia-76ers-vs-new-york-knicks-4/"
	for target, want := range map[string]bool{
		"https://v2.streameast.ga" + path + "2":                         true,
		"https://v2.streameast.ga/connect.php?redirect=%2Fnba%2F":       true,
		"https://auth.streamea.st/SsoHandoff.php?h=v2.streameast.ga":    true,
		"https://v2.streameast.ga/nhl/boston-bruins-vs-dallas-stars-1/": false,
		"https://v2.streameast.ga/buy-premium/":                         false,
		"https://ll.reliedhounder.com/x":                                false,
		"itms-apps://apps.apple.com/app/1":                              false,
	} {
		if got := allowedNavigation(site, path, target); got != want {
			t.Errorf("allowedNavigation(%q) = %v, want %v", target, got, want)
		}
	}
}

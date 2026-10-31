package main

import (
	"net/url"
	"os"
	"testing"

	"golang.org/x/net/html"
)

func TestParseStreamseasteGames(t *testing.T) {
	f, err := os.Open("testdata/streamseaste-nba.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	doc, err := html.Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	final, _ := url.Parse("https://streamseaste.cx/nba")
	scores := map[string]streamseasteScore{
		normalizeTeam("Boston Celtics") + "|" + normalizeTeam("Chicago Bulls"): {
			homeScore: "125",
			awayScore: "111",
			status:    "4th",
			live:      true,
		},
	}
	// At now = 1793404000, Celtics vs Bulls is ongoing (4th quarter)
	games := parseStreamseasteGamesAt(doc, final, scores, 1793404000)

	// Verify Celtics vs Bulls with score
	var celticsGame *Game
	for i := range games {
		if games[i].Home.Name == "Boston Celtics" {
			celticsGame = &games[i]
			break
		}
	}
	if celticsGame == nil {
		t.Fatal("Boston Celtics game not found")
	}
	if celticsGame.Home.Short != "Celtics" {
		t.Errorf("Home.Short = %q, want %q", celticsGame.Home.Short, "Celtics")
	}
	if celticsGame.Away.Short != "Bulls" {
		t.Errorf("Away.Short = %q, want %q", celticsGame.Away.Short, "Bulls")
	}
	if celticsGame.Home.Score != "125" {
		t.Errorf("Home.Score = %q, want %q", celticsGame.Home.Score, "125")
	}
	if celticsGame.Away.Score != "111" {
		t.Errorf("Away.Score = %q, want %q", celticsGame.Away.Score, "111")
	}
	if celticsGame.Status != "4th" {
		t.Errorf("Status = %q, want %q", celticsGame.Status, "4th")
	}
	if !celticsGame.Live {
		t.Error("expected celticsGame to be Live")
	}

	// Verify that finished games are removed
	finishedScores := map[string]streamseasteScore{
		normalizeTeam("Boston Celtics") + "|" + normalizeTeam("Chicago Bulls"): {
			homeScore: "125",
			awayScore: "111",
			status:    "Final",
			live:      false,
		},
	}
	// At later time 1793414784, Celtics vs Bulls is marked Final
	laterGames := parseStreamseasteGamesAt(doc, final, finishedScores, 1793414784)
	for _, g := range laterGames {
		if g.Home.Name == "Boston Celtics" {
			t.Errorf("expected finished Boston Celtics game to be removed")
		}
	}
}

func TestFindProvider(t *testing.T) {
	cases := []struct {
		input  string
		wantID string
	}{
		{"streamseaste.cx", "streamseaste"},
		{"https://streamseaste.cx/nba", "streamseaste"},
		{"v2.streameast.ga", "streameast"},
		{"https://streameast.sk", "streameast"},
		{"streameast.is", "streameast"},
	}
	for _, tc := range cases {
		p := findProvider(tc.input)
		if p.ID() != tc.wantID {
			t.Errorf("findProvider(%q).ID() = %q, want %q", tc.input, p.ID(), tc.wantID)
		}
	}
}

func TestLiveStreamseaste(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live test in short mode")
	}
	p := &StreamseasteProvider{}
	games, origin, err := p.FetchGames(t.Context(), "https://streamseaste.cx")
	if err != nil {
		t.Logf("live fetch failed: %v", err)
		return
	}
	if origin != "https://streamseaste.cx" {
		t.Errorf("origin = %q, want https://streamseaste.cx", origin)
	}
	if len(games) == 0 {
		t.Error("expected at least 1 game from live streamseaste")
	}
	t.Logf("successfully fetched %d games from live streamseaste", len(games))
}

package main

import (
	"testing"
	"time"
)

func TestESPNTime(t *testing.T) {
	for raw, want := range map[string]time.Time{
		"2026-10-31T19:00Z":    time.Date(2026, 10, 31, 19, 0, 0, 0, time.UTC),
		"2026-10-31T19:00:30Z": time.Date(2026, 10, 31, 19, 0, 30, 0, time.UTC),
	} {
		got, err := espnTime(raw)
		if err != nil || !got.Equal(want) {
			t.Errorf("espnTime(%q) = %v, %v; want %v", raw, got, err, want)
		}
	}
	if _, err := espnTime("tomorrow"); err == nil {
		t.Error("espnTime accepted a date it cannot read")
	}
}

// finalEvent is Cleveland 98, Oklahoma City 101, finished.
func finalEvent(id string) espnEvent {
	var e espnEvent
	e.ID = id
	e.Date = "2026-11-02T19:00Z"
	e.Start, _ = espnTime(e.Date)
	e.Competitions = []espnCompetition{{
		Competitors: []espnCompetitor{
			{
				HomeAway: "home", Score: "98",
				Team: espnTeam{Abbreviation: "CLE", DisplayName: "Cleveland Cavaliers", ShortDisplayName: "Cavaliers"},
			},
			{
				HomeAway: "away", Score: "101", Winner: true,
				Team: espnTeam{Abbreviation: "OKC", DisplayName: "Oklahoma City Thunder", ShortDisplayName: "Thunder"},
			},
		},
	}}
	e.Competitions[0].Status.Type.State = "post"
	e.Competitions[0].Status.Type.Completed = true
	e.Competitions[0].Status.Type.Detail = "Final"
	return e
}

func TestMatchEvent(t *testing.T) {
	events := []espnEvent{finalEvent("401")}
	game := Game{
		Home: Team{Name: "Cleveland Cavaliers", Short: "Cavaliers"},
		Away: Team{Name: "Oklahoma City Thunder", Short: "Thunder"},
	}
	if e, ok := matchEvent(events, game); !ok || e.ID != "401" {
		t.Fatalf("matchEvent by teams = %q, %v; want the event", e.ID, ok)
	}
	// The ESPN ID wins even when the names are not the site's.
	byID := Game{ESPNID: "401", Home: Team{Name: "Cavs"}, Away: Team{Name: "Thunder"}}
	if e, ok := matchEvent(events, byID); !ok || e.ID != "401" {
		t.Fatalf("matchEvent by ID = %q, %v; want the event", e.ID, ok)
	}
	// The home and away teams must be on the right sides.
	swapped := Game{
		Home: Team{Name: "Oklahoma City Thunder", Short: "Thunder"},
		Away: Team{Name: "Cleveland Cavaliers", Short: "Cavaliers"},
	}
	if e, ok := matchEvent(events, swapped); ok {
		t.Errorf("matchEvent matched swapped teams to %q", e.ID)
	}
}

func TestMergeEvents(t *testing.T) {
	site := Game{
		ID:   "47990",
		Path: "/nba/oklahoma-city-thunder-vs-cleveland-cavaliers-1/",
		Home: Team{Name: "Cleveland Cavaliers", Short: "Cavaliers"},
		Away: Team{Name: "Oklahoma City Thunder", Short: "Thunder"},
		Live: true, Status: "4th",
	}
	games := mergeEvents([]espnEvent{finalEvent("401")}, []Game{site})
	if len(games) != 1 {
		t.Fatalf("got %d games, want the site's one", len(games))
	}
	g := games[0]
	if !g.Final || g.Live || g.Status != "Final" || g.ESPNID != "401" {
		t.Errorf("finished game = %+v; want it final, on ESPN's ID", g)
	}
	if g.Home.Score != "98" || g.Away.Score != "101" {
		t.Errorf("scores = %s-%s, want 98-101", g.Home.Score, g.Away.Score)
	}

	// A game the site no longer lists is added from ESPN.
	games = mergeEvents([]espnEvent{finalEvent("402")}, nil)
	if len(games) != 1 || games[0].ID != "402" || !games[0].Final {
		t.Fatalf("finished game missing from the list: %+v", games)
	}
	if games[0].Home.Name != "Cleveland Cavaliers" || games[0].Away.Score != "101" {
		t.Errorf("finished game = %+v; want ESPN's teams and score", games[0])
	}
}

func TestApplyEventStatus(t *testing.T) {
	for _, tt := range []struct {
		name, state, detail, oldStatus string
		wantLive, wantFinal            bool
		wantStatus                     string
	}{
		{"live period", "in", "3:42 - 4th Quarter", "Halftime", true, false, "3:42 - 4th Quarter"},
		{"missing live period", "in", "", "Halftime", true, false, ""},
		{"pregame", "pre", "7:00 PM ET", "Final", false, false, "7:00 PM ET"},
		{"final overtime", "post", "Final/OT", "4th", false, true, "Final/OT"},
		{"missing final detail", "post", "", "4th", false, true, "Final"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := finalEvent("401")
			e.Competitions[0].Status.Type.State = tt.state
			e.Competitions[0].Status.Type.Completed = tt.wantFinal
			e.Competitions[0].Status.Type.Detail = tt.detail
			g := Game{Status: tt.oldStatus}
			applyEvent(&g, e)
			if g.Live != tt.wantLive || g.Final != tt.wantFinal || g.Status != tt.wantStatus {
				t.Fatalf("merged state = live:%v final:%v status:%q; want live:%v final:%v status:%q", g.Live, g.Final, g.Status, tt.wantLive, tt.wantFinal, tt.wantStatus)
			}
			if tt.wantLive && (g.Home.Score != "98" || g.Away.Score != "101") {
				t.Fatalf("live scores = %s-%s; want 98-101", g.Home.Score, g.Away.Score)
			}
		})
	}
}

func TestSortGames(t *testing.T) {
	games := []Game{
		{ID: "final", Final: true, Start: 100},
		{ID: "upcoming", Start: 300},
		{ID: "live", Live: true, Start: 200},
	}
	sortGames(games)
	want := []string{"live", "upcoming", "final"}
	for i, id := range want {
		if games[i].ID != id {
			t.Errorf("games[%d] = %q, want %q", i, games[i].ID, id)
		}
	}
}

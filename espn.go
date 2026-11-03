package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

// eastern is the time zone ESPN schedules and dates its games by.
var eastern = func() *time.Location {
	if loc, err := time.LoadLocation("America/New_York"); err == nil {
		return loc
	}
	return time.FixedZone("EDT", -4*3600)
}()

const (
	scoreboardURL = "https://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard?dates=%s"
	summaryURL    = "https://site.api.espn.com/apis/site/v2/sports/basketball/nba/summary?event=%s"
)

// espnStatus is where a game stands.
type espnStatus struct {
	Period int `json:"period"`
	Type   struct {
		State     string `json:"state"`     // "pre", "in", "post"
		Completed bool   `json:"completed"` // the game is over
		Detail    string `json:"detail"`    // "Final", "Final/OT", "4th Qtr"
	} `json:"type"`
}

// isFinal tells whether the game ended.
func (s espnStatus) isFinal() bool {
	return s.Type.Completed || s.Type.State == "post"
}

// espnTeam is a team as ESPN gives it.
type espnTeam struct {
	Abbreviation     string `json:"abbreviation"`
	DisplayName      string `json:"displayName"`
	ShortDisplayName string `json:"shortDisplayName"`
	Logo             string `json:"logo"`  // on a scoreboard; a summary lists "logos"
	Color            string `json:"color"` // "4e008e", without the #
}

// espnLine is one period's score.
type espnLine struct {
	DisplayValue string `json:"displayValue"`
}

// espnRecord is a team's win-loss record.
type espnRecord struct {
	Type    string `json:"type"` // "total" in a summary, "overall" on a scoreboard
	Summary string `json:"summary"`
}

// espnCompetitor is one side of a game.
type espnCompetitor struct {
	HomeAway   string       `json:"homeAway"`
	Winner     bool         `json:"winner"`
	Score      string       `json:"score"`
	Team       espnTeam     `json:"team"`
	Linescores []espnLine   `json:"linescores"`
	Record     []espnRecord `json:"record"`  // a summary's name for the records
	Records    []espnRecord `json:"records"` // a scoreboard's name for them
}

// record returns the team's overall record, e.g. "6-1".
func (c espnCompetitor) record() string {
	for _, r := range append(c.Record, c.Records...) {
		if r.Type == "total" || r.Type == "overall" {
			return r.Summary
		}
	}
	return ""
}

// espnBroadcast lists the networks showing a game.
type espnBroadcast struct {
	Names []string `json:"names"`
}

// espnCompetition is one game of an event; ESPN lists exactly one.
type espnCompetition struct {
	Date        string           `json:"date"` // "2026-10-31T19:00Z"
	Status      espnStatus       `json:"status"`
	Competitors []espnCompetitor `json:"competitors"`
	Broadcasts  []espnBroadcast  `json:"broadcasts"`
}

// espnEvent is a game on ESPN's scoreboard.
type espnEvent struct {
	ID           string            `json:"id"`
	Date         string            `json:"date"` // "2026-10-31T19:00Z"
	Competitions []espnCompetition `json:"competitions"`
	Start        time.Time         `json:"-"` // Date, parsed
}

// comp returns the event's game.
func (e espnEvent) comp() (espnCompetition, bool) {
	if len(e.Competitions) == 0 {
		return espnCompetition{}, false
	}
	return e.Competitions[0], true
}

// status returns where the event's game stands.
func (e espnEvent) status() espnStatus {
	c, _ := e.comp()
	return c.Status
}

// scoreboardDays caches the days fetched while the app runs, as the game
// list asks for them again every minute.
var scoreboardDays = struct {
	sync.Mutex
	day map[string]espnDay
}{day: map[string]espnDay{}}

type espnDay struct {
	events []espnEvent
	at     time.Time
}

// espnScoreboard lists the games of date, a US Eastern date such as
// "20261103". It is cached briefly: scores change during a game.
func espnScoreboard(ctx context.Context, date string) ([]espnEvent, error) {
	scoreboardDays.Lock()
	if day, ok := scoreboardDays.day[date]; ok && time.Since(day.at) < 45*time.Second {
		events := day.events
		scoreboardDays.Unlock()
		return events, nil
	}
	scoreboardDays.Unlock()

	events, err := fetchScoreboard(ctx, date)
	if err != nil {
		return nil, err
	}
	scoreboardDays.Lock()
	for d, day := range scoreboardDays.day {
		if time.Since(day.at) > time.Hour {
			delete(scoreboardDays.day, d)
		}
	}
	scoreboardDays.day[date] = espnDay{events: events, at: time.Now()}
	scoreboardDays.Unlock()
	return events, nil
}

// espnTime parses ESPN's dates, "2026-10-31T19:00Z": minutes, not seconds.
func espnTime(s string) (time.Time, error) {
	for _, layout := range []string{"2006-01-02T15:04Z07:00", time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("ESPN gave an unreadable date %q", s)
}

func fetchScoreboard(ctx context.Context, date string) ([]espnEvent, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(scoreboardURL, date), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ESPN answered %s", resp.Status)
	}
	var board struct {
		Events []espnEvent `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&board); err != nil {
		return nil, err
	}
	for i := range board.Events {
		if t, err := espnTime(board.Events[i].Date); err == nil {
			board.Events[i].Start = t
		}
	}
	return board.Events, nil
}

// espnToday lists the games that start today, by the user's own clock. A
// game on today's date may be on any of three US Eastern dates, depending on
// the user's time zone.
func espnToday(ctx context.Context) []espnEvent {
	// The site may answer slowly; a missing scoreboard only costs the
	// finished games, not the list.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	today := time.Now()
	var dates []string
	for d := -1; d <= 1; d++ {
		dates = append(dates, today.AddDate(0, 0, d).In(eastern).Format("20060102"))
	}

	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		events []espnEvent
	)
	for _, date := range dates {
		wg.Go(func() {
			day, err := espnScoreboard(ctx, date)
			if err != nil {
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, e := range day {
				if sameDay(e.Start, today) {
					events = append(events, e)
				}
			}
		})
	}
	wg.Wait()
	return events
}

// sameDay tells whether a and b fall on the same date, where the user is.
func sameDay(a, b time.Time) bool {
	a, b = a.In(time.Local), b.In(time.Local)
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// mergeToday takes the live state and scores of today's games from ESPN,
// and keeps the finished games the site no longer lists.
func mergeToday(ctx context.Context, games []Game) []Game {
	return mergeEvents(espnToday(ctx), games)
}

func mergeEvents(events []espnEvent, games []Game) []Game {
	seen := map[string]bool{}
	for i := range games {
		if e, ok := matchEvent(events, games[i]); ok {
			seen[e.ID] = true
			applyEvent(&games[i], e)
		}
	}
	for _, e := range events {
		if seen[e.ID] || !e.status().isFinal() {
			continue
		}
		if g, ok := gameFromEvent(e); ok {
			games = append(games, g)
		}
	}
	sortGames(games)
	return games
}

// matchEvent finds the ESPN event of game, by its ESPN ID or, for a page
// from an older mirror, by its teams.
func matchEvent(events []espnEvent, game Game) (espnEvent, bool) {
	if game.ESPNID != "" {
		for _, e := range events {
			if e.ID == game.ESPNID {
				return e, true
			}
		}
	}
	if game.Home.Name == "" || game.Away.Name == "" {
		return espnEvent{}, false
	}
	for _, e := range events {
		c, ok := e.comp()
		if !ok || len(c.Competitors) != 2 {
			continue
		}
		var home, away espnCompetitor
		for _, x := range c.Competitors {
			if x.HomeAway == "home" {
				home = x
			} else {
				away = x
			}
		}
		if (sameTeam(game.Home, home) && sameTeam(game.Away, away)) ||
			(sameTeam(game.Home, away) && sameTeam(game.Away, home)) {
			return e, true
		}
	}
	return espnEvent{}, false
}

// sameTeam tells whether a team of the site is the ESPN competitor.
func sameTeam(t Team, c espnCompetitor) bool {
	if code, ok := espnCodes[strings.ToLower(strings.TrimSpace(t.Name))]; ok {
		return strings.EqualFold(code, c.Team.Abbreviation)
	}
	short := t.Short
	if short == "" {
		short = teamShort(t.Name)
	}
	return strings.EqualFold(short, c.Team.ShortDisplayName)
}

// applyEvent updates game with where ESPN says it stands.
func applyEvent(game *Game, e espnEvent) {
	if game.ESPNID == "" {
		game.ESPNID = e.ID
	}
	c, _ := e.comp()
	// The site's home/away labels can be reversed. Align whole teams with
	// ESPN before assigning scores and colors, so they stay with the team.
	for _, x := range c.Competitors {
		if (x.HomeAway == "home" && sameTeam(game.Away, x)) ||
			(x.HomeAway == "away" && sameTeam(game.Home, x)) {
			game.Home, game.Away = game.Away, game.Home
			break
		}
	}
	s := e.status()
	game.Live = s.Type.State == "in"
	game.Final = s.isFinal()
	if game.Final {
		// ESPN knows "Final" from "Final/OT".
		game.Status = s.Type.Detail
		if game.Status == "" {
			game.Status = "Final"
		}
	} else {
		// Keep the period in sync with the state and scores from ESPN.
		game.Status = s.Type.Detail
	}
	if game.Final || game.Live {
		for _, x := range c.Competitors {
			t := &game.Away
			if x.HomeAway == "home" {
				t = &game.Home
			}
			if x.Score != "" {
				t.Score = x.Score
			}
			if t.Color == "" {
				t.Color = espnColor(x.Team.Color)
			}
		}
	}
}

// gameFromEvent builds the card of a finished game the site no longer lists.
func gameFromEvent(e espnEvent) (Game, bool) {
	c, ok := e.comp()
	if !ok {
		return Game{}, false
	}
	g := Game{ID: e.ID, ESPNID: e.ID, Start: e.Start.Unix(), Final: true, Status: "Final"}
	if detail := e.status().Type.Detail; detail != "" {
		g.Status = detail
	}
	for _, x := range c.Competitors {
		t := Team{
			Name:  x.Team.DisplayName,
			Short: x.Team.ShortDisplayName,
			Logo:  x.Team.Logo,
			Score: x.Score,
			Color: espnColor(x.Team.Color),
		}
		if t.Short == "" {
			t.Short = teamShort(t.Name)
		}
		t.setLogos(x.Team.Abbreviation)
		if x.HomeAway == "home" {
			g.Home = t
		} else {
			g.Away = t
		}
	}
	return g, g.Home.Name != "" && g.Away.Name != ""
}

// sortGames puts the live games first, then today's games still to come,
// and the finished ones last, newest first.
func sortGames(games []Game) {
	rank := func(g Game) int {
		switch {
		case g.Live:
			return 0
		case g.Final:
			return 2
		}
		return 1
	}
	slices.SortStableFunc(games, func(a, b Game) int {
		ra, rb := rank(a), rank(b)
		if ra != rb {
			return ra - rb
		}
		if ra == 2 {
			return int(b.Start - a.Start)
		}
		return int(a.Start - b.Start)
	})
}

// espnColor turns ESPN's "4e008e" into "#4e008e".
func espnColor(c string) string {
	if c == "" || strings.HasPrefix(c, "#") {
		return c
	}
	return "#" + c
}

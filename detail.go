package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// GameDetail is how a game ended: its line score, team stats and top
// performers, as ESPN has them.
type GameDetail struct {
	ID          string       `json:"id"`
	Start       int64        `json:"start"`  // Unix seconds
	Status      string       `json:"status"` // "Final", "Final/OT"
	Home        Team         `json:"home"`
	Away        Team         `json:"away"`
	HomeLine    []string     `json:"homeLine"` // the score of each period
	AwayLine    []string     `json:"awayLine"`
	Periods     []string     `json:"periods"`    // "1", "2", "3", "4", "OT"
	HomeRecord  string       `json:"homeRecord"` // "3-4"
	AwayRecord  string       `json:"awayRecord"`
	HomeWon     bool         `json:"homeWon"`
	AwayWon     bool         `json:"awayWon"`
	Venue       string       `json:"venue"`
	Stats       []DetailStat `json:"stats"`
	HomeLeaders []Leader     `json:"homeLeaders"`
	AwayLeaders []Leader     `json:"awayLeaders"`
}

// DetailStat is one row of the teams' side-by-side stats.
type DetailStat struct {
	Label     string  `json:"label"` // "FG"
	Home      string  `json:"home"`  // "40-86 (47%)"
	Away      string  `json:"away"`
	HomeValue float64 `json:"homeValue"` // for the comparison bar
	AwayValue float64 `json:"awayValue"`
}

// Leader is a team's best player in one category.
type Leader struct {
	Category string `json:"category"` // "Points"
	Name     string `json:"name"`
	Headshot string `json:"headshot"`
	Value    string `json:"value"`  // "27"
	Detail   string `json:"detail"` // "9/12 FG, 2/2 FT"
}

// Detail returns ESPN's account of a game: its line score, team stats and
// top performers.
func (n *NBA) Detail(ctx context.Context, game Game) (GameDetail, error) {
	id := game.ESPNID
	if id == "" {
		if e, ok := matchEvent(espnToday(ctx), game); ok {
			id = e.ID
		}
	}
	if id == "" {
		return GameDetail{}, errors.New("ESPN has no details for this game")
	}
	d, err := fetchSummary(ctx, id)
	if err != nil {
		return GameDetail{}, err
	}
	if d.Start == 0 {
		d.Start = game.Start
	}
	return d, nil
}

// statDefs are the team stats the detail page compares, in ESPN's order.
var statDefs = []struct {
	label string // "FG"
	name  string // the stat's name in ESPN's box score
	pct   string // the stat's percentage, or ""
}{
	{"FG", "fieldGoalsMade-fieldGoalsAttempted", "fieldGoalPct"},
	{"3PT", "threePointFieldGoalsMade-threePointFieldGoalsAttempted", "threePointFieldGoalPct"},
	{"FT", "freeThrowsMade-freeThrowsAttempted", "freeThrowPct"},
	{"REB", "totalRebounds", ""},
	{"AST", "assists", ""},
	{"STL", "steals", ""},
	{"BLK", "blocks", ""},
	{"TO", "totalTurnovers", ""},
	{"PITP", "pointsInPaint", ""},
}

// leaderCategories are the top performers the detail page shows.
var leaderCategories = []string{"points", "rebounds", "assists"}

func fetchSummary(ctx context.Context, id string) (GameDetail, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf(summaryURL, id), nil)
	if err != nil {
		return GameDetail{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return GameDetail{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return GameDetail{}, fmt.Errorf("ESPN answered %s", resp.Status)
	}
	var s struct {
		Header struct {
			Competitions []espnCompetition `json:"competitions"`
		} `json:"header"`
		Boxscore struct {
			Teams []struct {
				Team       espnTeam `json:"team"`
				Statistics []struct {
					Name         string `json:"name"`
					DisplayValue string `json:"displayValue"`
				} `json:"statistics"`
			} `json:"teams"`
		} `json:"boxscore"`
		Leaders []struct {
			Team    espnTeam `json:"team"`
			Leaders []struct {
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
				Leaders     []struct {
					DisplayValue string `json:"displayValue"`
					Summary      string `json:"summary"`
					Athlete      struct {
						DisplayName string `json:"displayName"`
						Headshot    struct {
							Href string `json:"href"`
						} `json:"headshot"`
					} `json:"athlete"`
				} `json:"leaders"`
			} `json:"leaders"`
		} `json:"leaders"`
		GameInfo struct {
			Venue struct {
				FullName string `json:"fullName"`
			} `json:"venue"`
		} `json:"gameInfo"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return GameDetail{}, err
	}
	if len(s.Header.Competitions) == 0 {
		return GameDetail{}, errors.New("ESPN has no details for this game")
	}
	c := s.Header.Competitions[0]
	d := GameDetail{ID: id, Status: c.Status.Type.Detail, Venue: s.GameInfo.Venue.FullName}
	if d.Status == "" && c.Status.isFinal() {
		d.Status = "Final"
	}
	if t, err := espnTime(c.Date); err == nil {
		d.Start = t.Unix()
	}

	var home, away string
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
		line := make([]string, 0, len(x.Linescores))
		for _, l := range x.Linescores {
			line = append(line, l.DisplayValue)
		}
		if x.HomeAway == "home" {
			d.Home, d.HomeLine, d.HomeRecord, d.HomeWon = t, line, x.record(), x.Winner
			home = strings.ToUpper(x.Team.Abbreviation)
		} else {
			d.Away, d.AwayLine, d.AwayRecord, d.AwayWon = t, line, x.record(), x.Winner
			away = strings.ToUpper(x.Team.Abbreviation)
		}
	}

	periods := max(len(d.HomeLine), len(d.AwayLine))
	for i := range periods {
		p := strconv.Itoa(i + 1)
		switch {
		case i == 4:
			p = "OT"
		case i > 4:
			p = "OT" + strconv.Itoa(i-3)
		}
		d.Periods = append(d.Periods, p)
	}

	byTeam := map[string]map[string]string{}
	for _, t := range s.Boxscore.Teams {
		stats := map[string]string{}
		for _, st := range t.Statistics {
			stats[st.Name] = st.DisplayValue
		}
		byTeam[strings.ToUpper(t.Team.Abbreviation)] = stats
	}
	homeStats, awayStats := byTeam[home], byTeam[away]
	for _, def := range statDefs {
		hv, hok := homeStats[def.name]
		av, aok := awayStats[def.name]
		if !hok || !aok {
			continue
		}
		row := DetailStat{Label: def.label, Home: hv, Away: av}
		if def.pct != "" {
			if hp, ok := homeStats[def.pct]; ok {
				row.Home = fmt.Sprintf("%s (%s%%)", hv, hp)
				row.HomeValue, _ = strconv.ParseFloat(hp, 64)
			}
			if ap, ok := awayStats[def.pct]; ok {
				row.Away = fmt.Sprintf("%s (%s%%)", av, ap)
				row.AwayValue, _ = strconv.ParseFloat(ap, 64)
			}
		} else {
			row.HomeValue, _ = strconv.ParseFloat(hv, 64)
			row.AwayValue, _ = strconv.ParseFloat(av, 64)
		}
		d.Stats = append(d.Stats, row)
	}

	for _, tl := range s.Leaders {
		side := &d.AwayLeaders
		if strings.EqualFold(tl.Team.Abbreviation, home) {
			side = &d.HomeLeaders
		}
		for _, want := range leaderCategories {
			for _, cat := range tl.Leaders {
				if cat.Name != want || len(cat.Leaders) == 0 {
					continue
				}
				l := cat.Leaders[0]
				*side = append(*side, Leader{
					Category: cat.DisplayName,
					Name:     l.Athlete.DisplayName,
					Headshot: l.Athlete.Headshot.Href,
					Value:    l.DisplayValue,
					Detail:   l.Summary,
				})
			}
		}
	}
	return d, nil
}

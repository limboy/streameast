package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"
)

// ScheduledGame is an upcoming game from ESPN's schedule.
type ScheduledGame struct {
	ID         string   `json:"id"`
	Home       Team     `json:"home"`
	Away       Team     `json:"away"`
	Start      int64    `json:"start"`      // Unix seconds
	Broadcasts []string `json:"broadcasts"` // "ESPN", "TNT"
}

// scoreboardURL is ESPN's public scoreboard for one day, as a US Eastern date.
const scoreboardURL = "https://site.api.espn.com/apis/site/v2/sports/basketball/nba/scoreboard?dates=%s"

var eastern = func() *time.Location {
	if loc, err := time.LoadLocation("America/New_York"); err == nil {
		return loc
	}
	return time.FixedZone("EDT", -4*3600)
}()

var scheduleCache struct {
	sync.Mutex
	games []ScheduledGame
	at    time.Time
}

// FetchSchedule lists the games ESPN schedules from yesterday to days days
// ahead, US Eastern, which covers the user's own next days in any time
// zone. The page groups them by the user's dates.
func FetchSchedule(ctx context.Context, days int) ([]ScheduledGame, error) {
	scheduleCache.Lock()
	defer scheduleCache.Unlock()
	if time.Since(scheduleCache.at) < 10*time.Minute {
		return scheduleCache.games, nil
	}
	today := time.Now().In(eastern)
	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		games []ScheduledGame
		errs  []error
	)
	for d := -1; d <= days; d++ {
		date := today.AddDate(0, 0, d).Format("20060102")
		wg.Go(func() {
			day, err := fetchScoreboard(ctx, date)
			mu.Lock()
			defer mu.Unlock()
			games = append(games, day...)
			if err != nil {
				errs = append(errs, err)
			}
		})
	}
	wg.Wait()
	if len(games) == 0 && len(errs) > 0 {
		return nil, errs[0]
	}
	slices.SortFunc(games, func(a, b ScheduledGame) int { return int(a.Start - b.Start) })
	scheduleCache.games, scheduleCache.at = games, time.Now()
	return games, nil
}

func fetchScoreboard(ctx context.Context, date string) ([]ScheduledGame, error) {
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
		Events []struct {
			ID           string `json:"id"`
			Date         string `json:"date"` // "2026-10-22T23:00Z"
			Competitions []struct {
				Competitors []struct {
					HomeAway string `json:"homeAway"`
					Team     struct {
						DisplayName      string `json:"displayName"`
						ShortDisplayName string `json:"shortDisplayName"`
						Logo             string `json:"logo"`
					} `json:"team"`
				} `json:"competitors"`
				Broadcasts []struct {
					Names []string `json:"names"`
				} `json:"broadcasts"`
			} `json:"competitions"`
		} `json:"events"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&board); err != nil {
		return nil, err
	}
	var games []ScheduledGame
	for _, e := range board.Events {
		start, err := time.Parse("2006-01-02T15:04Z07:00", e.Date)
		if err != nil || len(e.Competitions) == 0 {
			continue
		}
		c := e.Competitions[0]
		g := ScheduledGame{ID: e.ID, Start: start.Unix(), Broadcasts: []string{}}
		for _, t := range c.Competitors {
			team := Team{Name: t.Team.DisplayName, Short: t.Team.ShortDisplayName, Logo: t.Team.Logo}
			if t.HomeAway == "home" {
				g.Home = team
			} else {
				g.Away = team
			}
		}
		for _, b := range c.Broadcasts {
			for _, n := range b.Names {
				if !slices.Contains(g.Broadcasts, n) {
					g.Broadcasts = append(g.Broadcasts, n)
				}
			}
		}
		games = append(games, g)
	}
	return games, nil
}

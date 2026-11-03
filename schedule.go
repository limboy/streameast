package main

import (
	"context"
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
			day, err := espnScoreboard(ctx, date)
			mu.Lock()
			defer mu.Unlock()
			for _, e := range day {
				if g, ok := scheduledGame(e); ok {
					games = append(games, g)
				}
			}
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

// scheduledGame turns an ESPN event into a row of the schedule.
func scheduledGame(e espnEvent) (ScheduledGame, bool) {
	c, ok := e.comp()
	if !ok || e.Start.IsZero() {
		return ScheduledGame{}, false
	}
	g := ScheduledGame{ID: e.ID, Start: e.Start.Unix(), Broadcasts: []string{}}
	for _, x := range c.Competitors {
		t := Team{
			Name:  x.Team.DisplayName,
			Short: x.Team.ShortDisplayName,
			Logo:  x.Team.Logo,
			Color: espnColor(x.Team.Color),
		}
		t.setLogos(x.Team.Abbreviation)
		if x.HomeAway == "home" {
			g.Home = t
		} else {
			g.Away = t
		}
	}
	for _, b := range c.Broadcasts {
		for _, n := range b.Names {
			if !slices.Contains(g.Broadcasts, n) {
				g.Broadcasts = append(g.Broadcasts, n)
			}
		}
	}
	return g, g.Home.Name != "" && g.Away.Name != ""
}

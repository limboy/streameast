package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
)

// defaultSite is where the app looks for streams until the user picks
// another domain; the site moves between domains often.
const defaultSite = "https://v2.streameast.ga"

// Settings are the user's choices, kept in settings.json in the user data
// directory.
type Settings struct {
	Site     string `json:"site"`               // an origin, e.g. "https://v2.streameast.ga"
	Provider string `json:"provider,omitempty"` // provider ID: "streameast"
	Server   int    `json:"server,omitempty"`   // the server last watched, 1 when 0
}

// NBA is the service the game list calls.
type NBA struct {
	mu       sync.Mutex
	settings Settings
}

func settingsPath() string {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "settings.json")
}

func loadSettings() Settings {
	s := Settings{Site: defaultSite}
	if data, err := os.ReadFile(settingsPath()); err == nil {
		json.Unmarshal(data, &s)
	}
	if s.Site == "" || strings.Contains(s.Site, "streamseaste") {
		s.Site = defaultSite
	}
	return s
}

func (n *NBA) save() {
	path := settingsPath()
	if path == "" {
		return
	}
	os.MkdirAll(filepath.Dir(path), 0o755)
	data, _ := json.MarshalIndent(n.settings, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		log.Println("saving settings:", err)
	}
}

func (n *NBA) site() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.settings.Site
}

// Settings returns the current settings.
func (n *NBA) Settings() Settings {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.settings
}

// SetSite checks that site, a domain or URL such as "streameast.ga", lists
// NBA games, and makes it the site the app uses.
func (n *NBA) SetSite(ctx context.Context, site string) (Settings, error) {
	site = strings.TrimSpace(site)
	if !strings.Contains(site, "://") {
		site = "https://" + site
	}
	u, err := url.Parse(site)
	if err != nil || u.Host == "" {
		return n.Settings(), errors.New("that doesn't look like a domain")
	}

	p := findProvider(u.Host)
	games, origin, err := p.FetchGames(ctx, u.Scheme+"://"+u.Host)
	if err != nil || len(games) == 0 {
		// If preferred provider failed, try other providers in case it's a mirror of another site
		for _, alt := range providers {
			if alt.ID() == p.ID() {
				continue
			}
			if g, o, e := alt.FetchGames(ctx, u.Scheme+"://"+u.Host); e == nil && len(g) > 0 {
				p = alt
				games = g
				origin = o
				err = nil
				break
			}
		}
	}
	if err != nil || len(games) == 0 {
		errMsg := "no NBA games found"
		if err != nil {
			errMsg = err.Error()
		}
		return n.Settings(), errors.New("couldn't load NBA games from " + u.Host + ": " + errMsg)
	}
	n.mu.Lock()
	n.settings.Site = origin
	n.settings.Provider = p.ID()
	n.save()
	n.mu.Unlock()
	return n.Settings(), nil
}

// Domain is a site that answered with NBA games.
type Domain struct {
	Site     string `json:"site"`     // the origin it ended at, e.g. "https://v2.streameast.ga"
	Host     string `json:"host"`     // "v2.streameast.ga"
	Provider string `json:"provider"` // "StreamEast"
	Latency  int    `json:"latency"`  // how long the NBA page took, in milliseconds
	Games    int    `json:"games"`
}

// Domains checks the known domains, and the current one, at once, and lists
// those that work, fastest first.
func (n *NBA) Domains(ctx context.Context) []Domain {
	// Working domains answer in a few seconds; dead ones hang until this.
	ctx, cancel := context.WithTimeout(ctx, 7*time.Second)
	defer cancel()

	candidates := []string{n.site()}
	for _, p := range providers {
		for _, host := range p.KnownSites() {
			candidates = append(candidates, "https://"+host)
		}
	}
	slices.Sort(candidates)
	candidates = slices.Compact(candidates)

	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		found = map[string]Domain{}
	)
	for _, site := range candidates {
		site := site
		wg.Go(func() {
			start := time.Now()
			p := findProvider(site)
			games, origin, err := p.FetchGames(ctx, site)
			if err != nil || len(games) == 0 {
				return
			}
			d := Domain{
				Site:     origin,
				Provider: p.Name(),
				Latency:  int(time.Since(start).Milliseconds()),
				Games:    len(games),
			}
			if u, err := url.Parse(origin); err == nil {
				d.Host = u.Host
			}
			mu.Lock()
			defer mu.Unlock()
			// Domains that redirect to the same site count once, at their
			// fastest.
			if old, ok := found[origin]; !ok || d.Latency < old.Latency {
				found[origin] = d
			}
		})
	}
	wg.Wait()
	domains := slices.Collect(maps.Values(found))
	slices.SortFunc(domains, func(a, b Domain) int { return a.Latency - b.Latency })
	if domains == nil {
		domains = []Domain{}
	}
	return domains
}

// Games lists today's NBA games — live, still to come, and finished — from
// the site and ESPN, live ones first.
func (n *NBA) Games(ctx context.Context) ([]Game, error) {
	site := n.site()
	p := findProvider(site)
	games, origin, err := p.FetchGames(ctx, site)
	if err != nil {
		return nil, err
	}
	// Follow the site when it redirects to a new domain.
	n.mu.Lock()
	if origin != n.settings.Site {
		n.settings.Site = origin
		n.settings.Provider = p.ID()
		n.save()
	}
	n.mu.Unlock()
	// The site drops a game when it ends; ESPN still has it, finished, with
	// its final score.
	games = mergeToday(ctx, games)
	if games == nil {
		games = []Game{}
	}
	return games, nil
}

// Schedule lists the games of the next few days, from ESPN.
func (n *NBA) Schedule(ctx context.Context) ([]ScheduledGame, error) {
	games, err := FetchSchedule(ctx, 3)
	if games == nil {
		games = []ScheduledGame{}
	}
	return games, err
}

// Watch opens the game in a player window, or brings its window forward
// when it is already open.
func (n *NBA) Watch(game Game) {
	s := n.Settings()
	openPlayer(s.Site, game, max(s.Server, 1), n.setServer)
}

// setServer remembers the server the user picked, which the next game
// opens on: some servers are down more often than others.
func (n *NBA) setServer(server int) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.settings.Server != server {
		n.settings.Server = server
		n.save()
	}
}

func main() {
	nba := &NBA{}
	mygo.Bind(nba)

	mygo.App.WhenReady(func() {
		nba.settings = loadSettings()
		mygo.NewWindow(mygo.WindowOptions{
			Title:                "NBA",
			Width:                920,
			Height:               680,
			MinWidth:             420,
			MinHeight:            360,
			BackgroundColor:      "light-dark(#f4f5f7, #0b0d12)", // the page's --bg in src/style.css
			StateKey:             "main",
			TitleBarStyle:        mygo.TitleBarHidden,
			TitleBarHeight:       40,
			TrafficLightPosition: &mygo.Point{X: 14, Y: 14},
			URL:                  "/",
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

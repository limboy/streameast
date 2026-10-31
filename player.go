package main

import (
	_ "embed"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/egoist/mygo"
)

// playerScript runs in the game's page before the site's own scripts: it
// blocks third-party scripts, leaves only the video on screen and adds a
// bar to switch servers.
//
//go:embed player.js
var playerScript string

var players = struct {
	sync.Mutex
	byPath map[string]*mygo.Window
}{byPath: map[string]*mygo.Window{}}

// openPlayer opens game on server, and calls picked with the server the
// user switches to.
func openPlayer(site string, game Game, server int, picked func(server int)) {
	players.Lock()
	defer players.Unlock()
	if win := players.byPath[game.Path]; win != nil {
		win.Show()
		win.Focus()
		return
	}
	siteURL, err := url.Parse(site)
	if err != nil {
		return
	}
	p := findProvider(siteURL.Host)
	gameURL := p.GameURL(site, game, server)
	config, _ := json.Marshal(map[string]string{"title": game.Away.Name + " @ " + game.Home.Name, "path": game.Path})
	win := mygo.NewWindow(mygo.WindowOptions{
		Title:           game.Title(),
		Width:           1280,
		Height:          760,
		MinWidth:        480,
		MinHeight:       300,
		BackgroundColor: "#000000",
		StateKey:        "player",
		// player.js draws the title bar, 40 px tall, with the servers in
		// it, above the video and clear of the player's own controls.
		TitleBarStyle:        mygo.TitleBarHidden,
		TitleBarHeight:       40,
		TrafficLightPosition: &mygo.Point{X: 14, Y: 14},
		URL:                  gameURL,
		Page: mygo.PageOptions{
			UserAgent:     userAgent,
			PreloadScript: "window.__NBA_PLAYER__=" + string(config) + ";\n" + playerScript,
		},
	})
	players.byPath[game.Path] = win
	win.OnClosed(func() {
		players.Lock()
		delete(players.byPath, game.Path)
		players.Unlock()
	})

	page := win.Page()
	// Keep the window title ours, not the site's.
	page.OnPageTitleUpdated(func(e *mygo.TitleEvent) { e.PreventDefault() })
	// Popups and new tabs are ads: drop them all.
	page.SetWindowOpenHandler(func(mygo.WindowOpenRequest) *mygo.WindowOptions { return nil })
	// Stay on this game's pages; the redirects ads try to the top frame
	// go nowhere.
	page.OnWillNavigate(func(e *mygo.NavigateEvent) {
		if !p.AllowedNavigation(siteURL, game.Path, e.URL) {
			e.PreventDefault()
		}
	})
	page.OnDidNavigate(func(rawURL string) {
		if n := serverOf(game.Path, rawURL); n > 0 {
			picked(n)
		}
	})
}

// serverOf returns the server a game page shows, 2 for "/nba/…/2", or 0.
func serverOf(gamePath, rawURL string) int {
	u, err := url.Parse(rawURL)
	if err != nil {
		return 0
	}
	rest, ok := strings.CutPrefix(u.Path, gamePath)
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(strings.Trim(rest, "/"))
	if err != nil || n < 1 {
		return 0
	}
	return n
}

// allowedNavigation tells whether the player window may go to target: the
// game's own pages, and the site's sign-in handoff, which it bounces
// through on a first visit.
func allowedNavigation(site *url.URL, gamePath, target string) bool {
	u, err := url.Parse(target)
	if err != nil {
		return false
	}
	switch {
	case u.Scheme == "about":
		return true
	case u.Scheme != "https" && u.Scheme != "http":
		return false
	case u.Host == site.Host:
		return strings.HasPrefix(u.Path, gamePath) || strings.HasPrefix(u.Path, "/connect.php")
	case u.Host == "streamea.st" || strings.HasSuffix(u.Host, ".streamea.st"):
		return true
	}
	// A new domain of the site, as its SSO hops between them.
	return (strings.Contains(u.Host, "streameast") || strings.Contains(u.Host, "streamseaste")) && strings.HasPrefix(u.Path, gamePath)
}

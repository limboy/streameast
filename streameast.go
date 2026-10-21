package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// userAgent makes both our requests and the player's page look like Safari:
// the site serves bare web views a reduced page.
const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15"

// client keeps the site's cookies: its first visit bounces through an SSO
// handoff that sets them.
var client = func() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar, Timeout: 20 * time.Second}
}()

// Team is one side of a game.
type Team struct {
	Name  string `json:"name"`  // "Philadelphia 76ers"
	Short string `json:"short"` // "76ers"
	Logo  string `json:"logo"`  // an absolute URL, or ""
	Score string `json:"score"` // "" before the game
}

// Game is an NBA game listed by the site.
type Game struct {
	ID     string `json:"id"`
	Path   string `json:"path"` // the game's page, "/nba/…/"
	Home   Team   `json:"home"`
	Away   Team   `json:"away"`
	Start  int64  `json:"start"`  // Unix seconds
	Live   bool   `json:"live"`   // being played now
	Status string `json:"status"` // the period of a live game, "4th", "Halftime", or ""
}

// Title is how windows name the game.
func (g Game) Title() string { return g.Away.Short + " @ " + g.Home.Short }

// fetch gets a page of the site, following its redirects, and returns the
// parsed document and the URL it ended at.
func fetch(ctx context.Context, rawURL string) (*html.Node, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("%s answered %s", resp.Request.URL.Host, resp.Status)
	}
	doc, err := html.Parse(io.LimitReader(resp.Body, 8<<20))
	return doc, resp.Request.URL, err
}

// FetchGames lists the NBA games on the site at base, an origin such as
// "https://v2.streameast.ga". It also returns the origin the site redirected
// to, as the site moves between domains.
func FetchGames(ctx context.Context, base string) ([]Game, string, error) {
	doc, final, err := fetch(ctx, strings.TrimRight(base, "/")+"/nba-streams/")
	if err != nil {
		return nil, "", err
	}
	return parseGames(doc, final), final.Scheme + "://" + final.Host, nil
}

// parseGames reads the game cards of the NBA page at final.
func parseGames(doc *html.Node, final *url.URL) []Game {
	abs := func(src string) string {
		if src == "" {
			return ""
		}
		if u, err := final.Parse(src); err == nil {
			return u.String()
		}
		return ""
	}
	var games []Game
	seen := map[string]bool{}
	for card := range find(doc, func(n *html.Node) bool { return hasClass(n, "m-card") }) {
		link := first(card, func(n *html.Node) bool { return hasClass(n, "m-card__link") })
		path := attr(link, "href")
		if !strings.HasPrefix(path, "/nba/") || strings.HasPrefix(path, "/nba/teams/") || seen[path] {
			continue
		}
		seen[path] = true
		g := Game{ID: attr(card, "data-match-id"), Path: path, Live: hasClass(card, "m-card--live")}
		g.Start, _ = strconv.ParseInt(attr(card, "data-time"), 10, 64)
		names := strings.SplitN(attr(card, "data-team-names"), "|", 2)
		for i, side := range []*Team{&g.Home, &g.Away} {
			if i < len(names) {
				side.Name = names[i]
			}
			cls := []string{"m-card__team--home", "m-card__team--away"}[i]
			team := first(card, func(n *html.Node) bool { return hasClass(n, cls) })
			side.Short = text(first(team, func(n *html.Node) bool { return hasClass(n, "m-card__name") }))
			side.Score = text(first(team, func(n *html.Node) bool { return hasClass(n, "m-card__score") }))
			side.Logo = abs(attr(first(team, func(n *html.Node) bool { return hasClass(n, "m-card__crest-img") }), "src"))
			if side.Short == "" {
				side.Short = side.Name
			}
		}
		g.Status = text(first(card, func(n *html.Node) bool { return hasClass(n, "m-card__pill-status") }))
		games = append(games, g)
	}
	// Live games first, then by start time.
	slices.SortStableFunc(games, func(a, b Game) int {
		if a.Live != b.Live {
			if a.Live {
				return -1
			}
			return 1
		}
		return int(a.Start - b.Start)
	})
	return games
}

// find yields the nodes under n that match, in document order.
func find(n *html.Node, match func(*html.Node) bool) func(yield func(*html.Node) bool) {
	return func(yield func(*html.Node) bool) {
		if n == nil {
			return
		}
		for d := range n.Descendants() {
			if d.Type == html.ElementNode && match(d) && !yield(d) {
				return
			}
		}
	}
}

func first(n *html.Node, match func(*html.Node) bool) *html.Node {
	for d := range find(n, match) {
		return d
	}
	return nil
}

func attr(n *html.Node, key string) string {
	if n == nil {
		return ""
	}
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, class string) bool {
	return slices.Contains(strings.Fields(attr(n, "class")), class)
}

func text(n *html.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	for d := range n.Descendants() {
		if d.Type == html.TextNode {
			b.WriteString(d.Data)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

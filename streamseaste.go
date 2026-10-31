package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type streamseasteScore struct {
	homeScore string
	awayScore string
	status    string
	live      bool
}

func normalizeTeam(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func streamseasteStatus(st string) (label string, live bool) {
	switch strings.ToUpper(strings.TrimSpace(st)) {
	case "Q1":
		return "1st", true
	case "Q2":
		return "2nd", true
	case "Q3":
		return "3rd", true
	case "Q4":
		return "4th", true
	case "HT":
		return "Halftime", true
	case "OT":
		return "OT", true
	case "LIVE":
		return "LIVE", true
	case "FT":
		return "Final", false
	case "NS":
		return "", false
	default:
		return st, false
	}
}

// fetchLivescores fetches live scores from streamseaste's livescore_public.php
// if available, ignoring errors.
func fetchLivescores(ctx context.Context, base string) map[string]streamseasteScore {
	scores := map[string]streamseasteScore{}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", strings.TrimRight(base, "/")+"/livescore_public.php", nil)
	if err != nil {
		return scores
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return scores
	}
	defer resp.Body.Close()

	var data struct {
		Livescore []struct {
			StrHomeTeam  string `json:"strHomeTeam"`
			StrAwayTeam  string `json:"strAwayTeam"`
			IntHomeScore string `json:"intHomeScore"`
			IntAwayScore string `json:"intAwayScore"`
			StrStatus    string `json:"strStatus"`
		} `json:"livescore"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return scores
	}

	for _, item := range data.Livescore {
		h := normalizeTeam(item.StrHomeTeam)
		a := normalizeTeam(item.StrAwayTeam)
		if h == "" || a == "" {
			continue
		}
		label, live := streamseasteStatus(item.StrStatus)
		sc := streamseasteScore{
			homeScore: item.IntHomeScore,
			awayScore: item.IntAwayScore,
			status:    label,
			live:      live,
		}
		scores[h+"|"+a] = sc
		scores[a+"|"+h] = sc
	}
	return scores
}

// fetchStreamseasteGames loads games from streamseaste's /nba endpoint.
func fetchStreamseasteGames(ctx context.Context, base string) ([]Game, string, error) {
	doc, final, err := fetch(ctx, strings.TrimRight(base, "/")+"/nba")
	if err != nil {
		return nil, "", err
	}
	scores := fetchLivescores(ctx, base)
	games := parseStreamseasteGames(doc, final, scores)
	return games, final.Scheme + "://" + final.Host, nil
}

// parseStreamseasteGames parses game cards on the streamseaste page,
// omitting already finished games.
func parseStreamseasteGames(doc *html.Node, final *url.URL, scores map[string]streamseasteScore) []Game {
	return parseStreamseasteGamesAt(doc, final, scores, time.Now().Unix())
}

// parseStreamseasteGamesAt parses game cards with an explicit timestamp.
func parseStreamseasteGamesAt(doc *html.Node, final *url.URL, scores map[string]streamseasteScore, now int64) []Game {
	var games []Game
	seen := map[string]bool{}

	for card := range find(doc, func(n *html.Node) bool { return hasClass(n, "stream-card") }) {
		link := first(card, func(n *html.Node) bool { return hasClass(n, "card-link") })
		rawHref := attr(link, "href")
		if rawHref == "" {
			continue
		}
		u, err := final.Parse(rawHref)
		if err != nil || !strings.HasPrefix(u.Path, "/live/nba") || seen[u.Path] {
			continue
		}
		seen[u.Path] = true

		titleNode := first(card, func(n *html.Node) bool { return hasClass(n, "card-title") })
		titleText := text(titleNode)
		parts := strings.SplitN(titleText, " vs ", 2)
		if len(parts) != 2 {
			continue
		}

		homeName := strings.TrimSpace(parts[0])
		awayName := strings.TrimSpace(parts[1])

		startsStr := attr(card, "data-starts")
		endsStr := attr(card, "data-ends")
		if startsStr == "" {
			pill := first(card, func(n *html.Node) bool { return attr(n, "data-starts") != "" })
			if pill != nil {
				startsStr = attr(pill, "data-starts")
				if endsStr == "" {
					endsStr = attr(pill, "data-ends")
				}
			}
		}

		start, _ := strconv.ParseInt(startsStr, 10, 64)
		end, _ := strconv.ParseInt(endsStr, 10, 64)
		if end == 0 && start > 0 {
			end = start + 12000 // default NBA game duration ~3h20m
		}

		key := normalizeTeam(homeName) + "|" + normalizeTeam(awayName)
		sc, hasScore := scores[key]

		// Check if the game is already finished.
		// 1) livescore reports Final / FT
		// 2) scheduled end has passed and livescore is not reporting it as actively live
		isFinished := false
		if hasScore && sc.status == "Final" {
			isFinished = true
		} else if end > 0 && now > end && (!hasScore || !sc.live) {
			isFinished = true
		}
		if isFinished {
			continue
		}

		isLive := false
		if hasScore && sc.live {
			isLive = true
		} else if start > 0 && now >= start && (end == 0 || now <= end) {
			isLive = true
		}

		g := Game{
			ID:   path.Base(u.Path),
			Path: u.Path,
			Home: Team{
				Name:  homeName,
				Short: teamShort(homeName),
			},
			Away: Team{
				Name:  awayName,
				Short: teamShort(awayName),
			},
			Start: start,
			Live:  isLive,
		}
		g.Home.setLogos("")
		g.Away.setLogos("")

		// Enrich with real-time score and status from livescore if available
		if hasScore {
			if sc.homeScore != "" && sc.homeScore != "None" {
				g.Home.Score = sc.homeScore
			}
			if sc.awayScore != "" && sc.awayScore != "None" {
				g.Away.Score = sc.awayScore
			}
			if sc.status != "" {
				g.Status = sc.status
			}
		}

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

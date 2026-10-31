package main

import (
	"context"
	"net/url"
	"strconv"
	"strings"
)

// Provider abstracts a live sports streaming site.
type Provider interface {
	ID() string
	Name() string
	DefaultSite() string
	KnownSites() []string
	Matches(host string) bool
	FetchGames(ctx context.Context, base string) ([]Game, string, error)
	GameURL(site string, game Game, server int) string
	AllowedNavigation(site *url.URL, gamePath, target string) bool
}

// StreamEastProvider handles the official StreamEast network and its mirrors.
type StreamEastProvider struct{}

func (p *StreamEastProvider) ID() string   { return "streameast" }
func (p *StreamEastProvider) Name() string { return "StreamEast" }
func (p *StreamEastProvider) DefaultSite() string {
	return defaultSite
}
func (p *StreamEastProvider) KnownSites() []string {
	return []string{
		"v2.streameast.ga", "streameast.ga", "v2.streameast.ch", "streameast.ch",
		"streameast.sk", "streameast.app", "the-streameast.io", "streameast.to",
		"streameast.gd", "streameast.xyz", "streameast.gl", "thestreameast.co",
		"thestreameast.fun", "streameast.ec", "streameast.is",
	}
}
func (p *StreamEastProvider) Matches(host string) bool {
	host = strings.ToLower(host)
	if strings.Contains(host, "streameast") && !strings.Contains(host, "streamseaste") {
		return true
	}
	for _, s := range p.KnownSites() {
		if strings.EqualFold(host, s) {
			return true
		}
	}
	return false
}
func (p *StreamEastProvider) FetchGames(ctx context.Context, base string) ([]Game, string, error) {
	return FetchGames(ctx, base)
}
func (p *StreamEastProvider) GameURL(site string, game Game, server int) string {
	return strings.TrimRight(site, "/") + game.Path + strconv.Itoa(max(server, 1))
}
func (p *StreamEastProvider) AllowedNavigation(site *url.URL, gamePath, target string) bool {
	return allowedNavigation(site, gamePath, target)
}

// StreamseasteProvider handles streamseaste.cx and its mirrors.
type StreamseasteProvider struct{}

func (p *StreamseasteProvider) ID() string   { return "streamseaste" }
func (p *StreamseasteProvider) Name() string { return "Streamseaste" }
func (p *StreamseasteProvider) DefaultSite() string {
	return "https://streamseaste.cx"
}
func (p *StreamseasteProvider) KnownSites() []string {
	return []string{
		"streamseaste.cx",
	}
}
func (p *StreamseasteProvider) Matches(host string) bool {
	host = strings.ToLower(host)
	return strings.Contains(host, "streamseaste")
}
func (p *StreamseasteProvider) FetchGames(ctx context.Context, base string) ([]Game, string, error) {
	return fetchStreamseasteGames(ctx, base)
}
func (p *StreamseasteProvider) GameURL(site string, game Game, server int) string {
	return strings.TrimRight(site, "/") + game.Path
}
func (p *StreamseasteProvider) AllowedNavigation(site *url.URL, gamePath, target string) bool {
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
		return strings.HasPrefix(u.Path, gamePath)
	}
	return false
}

var providers = []Provider{
	&StreamseasteProvider{},
	&StreamEastProvider{},
}

// findProvider returns the Provider for a site or host.
func findProvider(siteOrHost string) Provider {
	host := siteOrHost
	if u, err := url.Parse(siteOrHost); err == nil && u.Host != "" {
		host = u.Host
	}
	for _, p := range providers {
		if p.Matches(host) {
			return p
		}
	}
	return providers[0]
}

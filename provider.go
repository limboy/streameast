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
	if strings.Contains(host, "streameast") {
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

var providers = []Provider{
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

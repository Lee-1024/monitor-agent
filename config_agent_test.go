package main

import "testing"

func TestEffectiveHostnamePrefersConfiguredHostname(t *testing.T) {
	config := &AgentConfig{Hostname: "display-name"}

	if got := config.EffectiveHostname("system-host"); got != "display-name" {
		t.Fatalf("expected configured hostname, got %q", got)
	}
}

func TestEffectiveHostnameFallsBackToSystemHostname(t *testing.T) {
	config := &AgentConfig{}

	if got := config.EffectiveHostname("system-host"); got != "system-host" {
		t.Fatalf("expected system hostname, got %q", got)
	}
}

func TestLogCollectionEnabledOnlyWhenLogPathsConfigured(t *testing.T) {
	if (&AgentConfig{}).LogCollectionEnabled() {
		t.Fatal("expected log collection to be disabled without log paths")
	}

	if !(&AgentConfig{LogPaths: []string{"/var/log/app.log"}}).LogCollectionEnabled() {
		t.Fatal("expected log collection to be enabled with log paths")
	}
}

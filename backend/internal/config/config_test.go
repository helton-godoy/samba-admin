package config

import "testing"

func TestFixtureModeRequiresJSONPath(t *testing.T) {
	if err := validateAdapterConfig(Config{AdapterMode: "fixture"}); err == nil {
		t.Fatal("modo fixture sem arquivo deveria ser recusado")
	}
	if err := validateAdapterConfig(Config{AdapterMode: "fixture", FixturePath: "fixture.txt"}); err == nil {
		t.Fatal("fixture sem extensão JSON deveria ser recusada")
	}
	if err := validateAdapterConfig(Config{AdapterMode: "fixture", FixturePath: "fixture.json"}); err != nil {
		t.Fatalf("fixture JSON válida foi recusada: %v", err)
	}
}

func TestUnknownAdapterModeFailsClosed(t *testing.T) {
	if err := validateAdapterConfig(Config{AdapterMode: "shell"}); err == nil {
		t.Fatal("adapter desconhecido deveria falhar fechado")
	}
}

func TestFixtureExecutorRequiresJSONPath(t *testing.T) {
	base := Config{
		AgentExecutorMode:    "fixture",
		AgentMaxClockSkew:    2,
		AgentNonceTTL:        2,
		AgentNonceCacheLimit: 128,
		AgentMaxMessageBytes: 1,
		AgentMaxOutputBytes:  1,
		AgentMaxConcurrent:   1,
		AgentSocketMode:      0o600,
		AgentExpectedPeerUID: -1,
	}
	if err := validateAgentConfig(base); err == nil {
		t.Fatal("executor fixture sem arquivo deveria ser recusado")
	}
	base.FixturePath = "fixture.txt"
	if err := validateAgentConfig(base); err == nil {
		t.Fatal("executor fixture sem extensão JSON deveria ser recusado")
	}
	base.FixturePath = "fixture.json"
	if err := validateAgentConfig(base); err != nil {
		t.Fatalf("executor fixture válido foi recusado: %v", err)
	}
}

func TestReadOnlyModesRejectMutableFeatureFlag(t *testing.T) {
	for _, cfg := range []Config{
		{AdapterMode: "freebsd-readonly", EnableMutableOperations: true},
		{AdapterMode: "fixture", FixturePath: "fixture.json", EnableMutableOperations: true},
		{AdapterMode: "agent", AgentExecutorMode: "fixture", FixturePath: "fixture.json", EnableMutableOperations: true},
		{AdapterMode: "agent", AgentExecutorMode: "readonly-freebsd", EnableMutableOperations: true},
	} {
		if err := validateAdapterConfig(cfg); err == nil {
			t.Fatalf("configuração somente leitura aceitou mutação: %+v", cfg)
		}
	}
}

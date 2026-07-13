package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr                string
	DatabasePath            string
	AgentSocket             string
	AgentSharedKey          string
	AgentPreviousSharedKey  string
	AgentKeyID              string
	AgentPreviousKeyID      string
	AgentNonceStorePath     string
	AgentNonceTTL           time.Duration
	AgentNonceCacheLimit    int
	AgentMaxMessageBytes    int64
	AgentMaxOutputBytes     int
	AgentMaxConcurrent      int
	AgentMaxClockSkew       time.Duration
	AgentSocketMode         os.FileMode
	AgentSocketOwner        string
	AgentSocketGroup        string
	AgentExpectedPeerUID    int
	AgentExecutorMode       string
	AgentCapabilities       []string
	AdapterMode             string
	FixturePath             string
	AuthMode                string
	AllowedOrigins          []string
	SessionTTL              time.Duration
	RequestTimeout          time.Duration
	MaxBodyBytes            int64
	BootstrapUser           string
	BootstrapPassword       string
	TOTPEncryptionKey       []byte
	MFARequiredRoles        []string
	EnableMutableOperations bool
	DevScenario             string
}

func Load() (Config, error) {
	dataDir := getenv("SAMBA_ADMIN_DATA_DIR", "./var")
	socketMode, err := socketModeValue("SAMBA_ADMIN_AGENT_SOCKET_MODE", 0o660)
	if err != nil {
		return Config{}, err
	}
	expectedPeerUID, err := intValueStrict("SAMBA_ADMIN_AGENT_EXPECTED_PEER_UID", -1)
	if err != nil {
		return Config{}, err
	}
	mutableOperations, err := boolValue("SAMBA_ADMIN_ENABLE_MUTABLE_OPERATIONS", false)
	if err != nil {
		return Config{}, err
	}
	totpKey, err := decodeOptionalBase64(os.Getenv("SAMBA_ADMIN_TOTP_ENCRYPTION_KEY"))
	if err != nil {
		return Config{}, err
	}
	agentNonceTTL, err := durationStrict("SAMBA_ADMIN_AGENT_NONCE_TTL", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	agentMaxClockSkew, err := durationStrict("SAMBA_ADMIN_AGENT_MAX_CLOCK_SKEW", 2*time.Minute)
	if err != nil {
		return Config{}, err
	}
	agentNonceCacheLimit, err := intValueStrict("SAMBA_ADMIN_AGENT_NONCE_CACHE_LIMIT", 4096)
	if err != nil {
		return Config{}, err
	}
	agentMaxOutputBytes, err := intValueStrict("SAMBA_ADMIN_AGENT_MAX_OUTPUT_BYTES", 128<<10)
	if err != nil {
		return Config{}, err
	}
	agentMaxConcurrent, err := intValueStrict("SAMBA_ADMIN_AGENT_MAX_CONCURRENT", 4)
	if err != nil {
		return Config{}, err
	}
	agentMaxMessageBytes, err := int64ValueStrict("SAMBA_ADMIN_AGENT_MAX_MESSAGE_BYTES", 1<<20)
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		HTTPAddr:                getenv("SAMBA_ADMIN_HTTP_ADDR", "127.0.0.1:8080"),
		DatabasePath:            getenv("SAMBA_ADMIN_DATABASE", filepath.Join(dataDir, "samba-admin.db")),
		AgentSocket:             getenv("SAMBA_ADMIN_AGENT_SOCKET", filepath.Join(dataDir, "samba-admin-agent.sock")),
		AgentSharedKey:          os.Getenv("SAMBA_ADMIN_AGENT_KEY"),
		AgentPreviousSharedKey:  os.Getenv("SAMBA_ADMIN_AGENT_PREVIOUS_KEY"),
		AgentKeyID:              getenv("SAMBA_ADMIN_AGENT_KEY_ID", "current"),
		AgentPreviousKeyID:      getenv("SAMBA_ADMIN_AGENT_PREVIOUS_KEY_ID", "previous"),
		AgentNonceStorePath:     getenv("SAMBA_ADMIN_AGENT_NONCE_STORE", filepath.Join(dataDir, "agent-nonces.json")),
		AgentNonceTTL:           agentNonceTTL,
		AgentNonceCacheLimit:    agentNonceCacheLimit,
		AgentMaxMessageBytes:    agentMaxMessageBytes,
		AgentMaxOutputBytes:     agentMaxOutputBytes,
		AgentMaxConcurrent:      agentMaxConcurrent,
		AgentMaxClockSkew:       agentMaxClockSkew,
		AgentSocketMode:         socketMode,
		AgentSocketOwner:        strings.TrimSpace(os.Getenv("SAMBA_ADMIN_AGENT_SOCKET_OWNER")),
		AgentSocketGroup:        strings.TrimSpace(os.Getenv("SAMBA_ADMIN_AGENT_SOCKET_GROUP")),
		AgentExpectedPeerUID:    expectedPeerUID,
		AgentExecutorMode:       getenv("SAMBA_ADMIN_AGENT_EXECUTOR", "mock"),
		AgentCapabilities:       split(os.Getenv("SAMBA_ADMIN_AGENT_CAPABILITIES")),
		AdapterMode:             getenv("SAMBA_ADMIN_ADAPTER_MODE", "mock"),
		FixturePath:             strings.TrimSpace(os.Getenv("SAMBA_ADMIN_FIXTURE_PATH")),
		AuthMode:                getenv("SAMBA_ADMIN_AUTH_MODE", "development-bypass"),
		AllowedOrigins:          split(getenv("SAMBA_ADMIN_ALLOWED_ORIGINS", "http://localhost:5173")),
		SessionTTL:              duration("SAMBA_ADMIN_SESSION_TTL", 8*time.Hour),
		RequestTimeout:          duration("SAMBA_ADMIN_REQUEST_TIMEOUT", 30*time.Second),
		MaxBodyBytes:            int64Value("SAMBA_ADMIN_MAX_BODY_BYTES", 2<<20),
		BootstrapUser:           getenv("SAMBA_ADMIN_BOOTSTRAP_USER", "admin"),
		BootstrapPassword:       os.Getenv("SAMBA_ADMIN_BOOTSTRAP_PASSWORD"),
		TOTPEncryptionKey:       totpKey,
		MFARequiredRoles:        split(os.Getenv("SAMBA_ADMIN_MFA_REQUIRED_ROLES")),
		EnableMutableOperations: mutableOperations,
		DevScenario:             getenv("SAMBA_ADMIN_DEV_SCENARIO", "success"),
	}
	if cfg.AgentSharedKey == "" {
		if cfg.AuthMode != "development-bypass" {
			return Config{}, errors.New("SAMBA_ADMIN_AGENT_KEY é obrigatório fora do modo de desenvolvimento")
		}
		buf := make([]byte, 32)
		_, _ = rand.Read(buf)
		cfg.AgentSharedKey = base64.RawURLEncoding.EncodeToString(buf)
	}
	if err := validateAgentConfig(cfg); err != nil {
		return Config{}, err
	}
	if err := validateAdapterConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
func getenv(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
func split(v string) []string {
	var out []string
	for _, p := range strings.Split(v, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
func duration(k string, d time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	x, e := time.ParseDuration(v)
	if e != nil {
		return d
	}
	return x
}

func durationStrict(k string, d time.Duration) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return d, nil
	}
	x, err := time.ParseDuration(v)
	if err != nil {
		return 0, errors.New(k + " deve ser uma duração válida")
	}
	return x, nil
}
func int64Value(k string, d int64) int64 {
	v := os.Getenv(k)
	if v == "" {
		return d
	}
	x, e := strconv.ParseInt(v, 10, 64)
	if e != nil {
		return d
	}
	return x
}

func int64ValueStrict(k string, d int64) (int64, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return d, nil
	}
	x, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, errors.New(k + " deve ser um inteiro")
	}
	return x, nil
}

func intValueStrict(k string, d int) (int, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return d, nil
	}
	x, err := strconv.Atoi(v)
	if err != nil {
		return 0, errors.New(k + " deve ser um inteiro")
	}
	return x, nil
}

func boolValue(k string, d bool) (bool, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return d, nil
	}
	x, err := strconv.ParseBool(v)
	if err != nil {
		return false, errors.New(k + " deve ser booleano")
	}
	return x, nil
}

func socketModeValue(k string, d os.FileMode) (os.FileMode, error) {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return d, nil
	}
	v = strings.TrimPrefix(v, "0o")
	v = strings.TrimPrefix(v, "0O")
	if v == "0" {
		return 0, nil
	}
	v = strings.TrimPrefix(v, "0")
	if v == "" {
		return 0, nil
	}
	x, err := strconv.ParseUint(v, 8, 32)
	if err != nil {
		return 0, errors.New(k + " deve ser um modo octal")
	}
	return os.FileMode(x), nil
}

func decodeOptionalBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	for _, encoding := range []*base64.Encoding{base64.RawStdEncoding, base64.StdEncoding, base64.RawURLEncoding, base64.URLEncoding} {
		decoded, err := encoding.DecodeString(value)
		if err == nil {
			if len(decoded) != 32 {
				return nil, errors.New("SAMBA_ADMIN_TOTP_ENCRYPTION_KEY deve decodificar para exatamente 32 bytes")
			}
			return decoded, nil
		}
	}
	return nil, errors.New("SAMBA_ADMIN_TOTP_ENCRYPTION_KEY deve ser base64 válido")
}

func validateAgentConfig(cfg Config) error {
	if cfg.AgentMaxClockSkew <= 0 || cfg.AgentMaxClockSkew > 10*time.Minute {
		return errors.New("SAMBA_ADMIN_AGENT_MAX_CLOCK_SKEW deve estar entre 1ns e 10m")
	}
	if cfg.AgentNonceTTL < cfg.AgentMaxClockSkew || cfg.AgentNonceTTL > 24*time.Hour {
		return errors.New("SAMBA_ADMIN_AGENT_NONCE_TTL deve cobrir a janela de validade e ser inferior a 24h")
	}
	if cfg.AgentNonceCacheLimit < 128 || cfg.AgentNonceCacheLimit > 100000 {
		return errors.New("SAMBA_ADMIN_AGENT_NONCE_CACHE_LIMIT deve estar entre 128 e 100000")
	}
	if cfg.AgentMaxMessageBytes < 1 || cfg.AgentMaxMessageBytes > 8<<20 {
		return errors.New("SAMBA_ADMIN_AGENT_MAX_MESSAGE_BYTES deve estar entre 1 e 8 MiB")
	}
	if cfg.AgentMaxOutputBytes < 1 || cfg.AgentMaxOutputBytes > 4<<20 {
		return errors.New("SAMBA_ADMIN_AGENT_MAX_OUTPUT_BYTES deve estar entre 1 e 4 MiB")
	}
	if cfg.AgentMaxConcurrent < 1 || cfg.AgentMaxConcurrent > 64 {
		return errors.New("SAMBA_ADMIN_AGENT_MAX_CONCURRENT deve estar entre 1 e 64")
	}
	if cfg.AgentExpectedPeerUID < -1 {
		return errors.New("SAMBA_ADMIN_AGENT_EXPECTED_PEER_UID deve ser -1 ou UID válido")
	}
	if cfg.AgentSocketMode&^os.FileMode(0o777) != 0 || cfg.AgentSocketMode&0o007 != 0 || cfg.AgentSocketMode&0o600 != 0o600 {
		return errors.New("SAMBA_ADMIN_AGENT_SOCKET_MODE deve ser restritivo e sem acesso para outros")
	}
	switch cfg.AgentExecutorMode {
	case "mock", "fixture", "readonly-freebsd":
	default:
		return errors.New("SAMBA_ADMIN_AGENT_EXECUTOR deve ser mock, fixture ou readonly-freebsd")
	}
	if cfg.AgentExecutorMode == "fixture" {
		if cfg.FixturePath == "" {
			return errors.New("SAMBA_ADMIN_AGENT_EXECUTOR=fixture exige SAMBA_ADMIN_FIXTURE_PATH")
		}
		if !strings.EqualFold(filepath.Ext(cfg.FixturePath), ".json") {
			return errors.New("SAMBA_ADMIN_FIXTURE_PATH deve apontar para arquivo .json")
		}
	}
	if cfg.AgentExecutorMode == "readonly-freebsd" {
		if cfg.AgentSocketOwner == "" || cfg.AgentSocketGroup == "" || cfg.AgentExpectedPeerUID < 0 {
			return errors.New("readonly-freebsd exige proprietário, grupo e UID esperado do peer para o socket")
		}
		if len(cfg.AgentCapabilities) == 0 {
			return errors.New("readonly-freebsd exige capabilities explicitamente habilitadas")
		}
	}
	return nil
}

func validateAdapterConfig(cfg Config) error {
	if cfg.EnableMutableOperations && (cfg.AdapterMode == "freebsd-readonly" || cfg.AdapterMode == "fixture" || cfg.AgentExecutorMode == "fixture" || cfg.AgentExecutorMode == "readonly-freebsd") {
		return errors.New("operações mutáveis não podem ser habilitadas em adapter ou executor somente leitura")
	}
	switch cfg.AdapterMode {
	case "mock", "agent", "freebsd-readonly":
		return nil
	case "fixture":
		if cfg.FixturePath == "" {
			return errors.New("SAMBA_ADMIN_FIXTURE_PATH é obrigatório no modo fixture")
		}
		if !strings.EqualFold(filepath.Ext(cfg.FixturePath), ".json") {
			return errors.New("SAMBA_ADMIN_FIXTURE_PATH deve apontar para arquivo .json")
		}
		return nil
	default:
		return errors.New("SAMBA_ADMIN_ADAPTER_MODE deve ser mock, fixture, agent ou freebsd-readonly")
	}
}

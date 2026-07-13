package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

var (
	ErrNonceReplay  = errors.New("nonce já utilizado")
	ErrInvalidNonce = errors.New("nonce inválido")
)

// NonceStore is a bounded replay cache. When Path is configured, entries are
// atomically persisted with restrictive permissions so restarting the agent
// cannot reopen the replay window.
type NonceStore struct {
	mu      sync.Mutex
	path    string
	ttl     time.Duration
	max     int
	entries map[string]time.Time
}

type nonceFile struct {
	Entries map[string]time.Time `json:"entries"`
}

func NewNonceStore(path string, ttl time.Duration, maxEntries int) (*NonceStore, error) {
	if ttl <= 0 {
		return nil, errors.New("TTL de nonce deve ser positivo")
	}
	if maxEntries < 128 {
		return nil, errors.New("limite de nonce deve ser ao menos 128")
	}
	store := &NonceStore{path: strings.TrimSpace(path), ttl: ttl, max: maxEntries, entries: map[string]time.Time{}}
	if store.path == "" {
		return store, nil
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	store.mu.Lock()
	changed := store.purgeLocked(time.Now().UTC())
	if changed {
		err := store.persistLocked()
		store.mu.Unlock()
		if err != nil {
			return nil, err
		}
		return store, nil
	}
	store.mu.Unlock()
	return store, nil
}

// Use records a nonce before work is scheduled. Failure to persist a configured
// store fails closed, preventing a request from running without replay state.
func (s *NonceStore) Use(nonce string, now time.Time) error {
	if err := validateNonce(nonce); err != nil {
		return err
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	previous := cloneNonceEntries(s.entries)
	s.purgeLocked(now)
	if _, exists := s.entries[nonce]; exists {
		return ErrNonceReplay
	}
	s.entries[nonce] = now.Add(s.ttl)
	s.trimLocked()
	if err := s.persistLocked(); err != nil {
		s.entries = previous
		return fmt.Errorf("persistência do cache de nonce: %w", err)
	}
	return nil
}

func validateNonce(nonce string) error {
	if len(nonce) < 16 || len(nonce) > 256 {
		return ErrInvalidNonce
	}
	for _, r := range nonce {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return ErrInvalidNonce
		}
	}
	return nil
}

func (s *NonceStore) load() error {
	if err := validateNonceDirectory(filepath.Dir(s.path), false); err != nil {
		return err
	}
	info, err := os.Lstat(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return errors.New("arquivo de nonce deve ser regular e não pode ser symlink")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return errors.New("arquivo de nonce possui permissões excessivas")
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	if len(data) > 8<<20 {
		return errors.New("arquivo de nonce excede o limite permitido")
	}
	var persisted nonceFile
	if err = json.Unmarshal(data, &persisted); err != nil {
		return fmt.Errorf("arquivo de nonce inválido: %w", err)
	}
	if len(persisted.Entries) > s.max {
		return errors.New("arquivo de nonce excede o limite de entradas")
	}
	for nonce, expiresAt := range persisted.Entries {
		if err := validateNonce(nonce); err != nil {
			return errors.New("arquivo de nonce contém entrada inválida")
		}
		s.entries[nonce] = expiresAt.UTC()
	}
	return nil
}

func (s *NonceStore) purgeLocked(now time.Time) bool {
	changed := false
	for nonce, expiresAt := range s.entries {
		if !expiresAt.After(now) {
			delete(s.entries, nonce)
			changed = true
		}
	}
	return changed
}

func (s *NonceStore) trimLocked() {
	if len(s.entries) <= s.max {
		return
	}
	type entry struct {
		nonce     string
		expiresAt time.Time
	}
	entries := make([]entry, 0, len(s.entries))
	for nonce, expiresAt := range s.entries {
		entries = append(entries, entry{nonce: nonce, expiresAt: expiresAt})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].expiresAt.Before(entries[j].expiresAt) })
	for i := 0; i < len(entries)-s.max; i++ {
		delete(s.entries, entries[i].nonce)
	}
}

func (s *NonceStore) persistLocked() error {
	if s.path == "" {
		return nil
	}
	dir := filepath.Dir(s.path)
	if err := validateNonceDirectory(dir, true); err != nil {
		return err
	}
	data, err := json.Marshal(nonceFile{Entries: s.entries})
	if err != nil {
		return err
	}
	if len(data) > 8<<20 {
		return errors.New("cache de nonce excede o tamanho permitido")
	}
	tmp, err := os.CreateTemp(dir, ".samba-admin-nonces-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err = tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = os.Rename(tmpName, s.path); err != nil {
		return err
	}
	// Synchronize the directory entry where the platform permits it.
	if dirFile, openErr := os.Open(dir); openErr == nil {
		_ = dirFile.Sync()
		_ = dirFile.Close()
	}
	return nil
}

func validateNonceDirectory(dir string, create bool) error {
	if create {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	info, err := os.Lstat(dir)
	if errors.Is(err, fs.ErrNotExist) && !create {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("diretório de nonce inválido")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return errors.New("diretório de nonce possui permissões excessivas")
	}
	return nil
}

func cloneNonceEntries(entries map[string]time.Time) map[string]time.Time {
	clone := make(map[string]time.Time, len(entries))
	for nonce, expiresAt := range entries {
		clone[nonce] = expiresAt
	}
	return clone
}

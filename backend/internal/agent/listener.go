package agent

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SocketOptions keeps deployment-specific Unix socket policy out of request
// handling. Owner and group accept numeric IDs or local account names.
type SocketOptions struct {
	Mode                   os.FileMode
	Owner                  string
	Group                  string
	ExpectedPeerUID        int
	EnforcePeerCredentials bool
}

// PrepareSocketDirectory creates the dedicated socket directory with a
// traversable group mode. Callers must not point it at a shared system
// directory such as /var/run itself.
func PrepareSocketDirectory(dir, owner, group string) error {
	clean := filepath.Clean(dir)
	if clean == "." || clean == string(filepath.Separator) || isSharedSystemDirectory(clean) {
		return errors.New("diretório dedicado do socket é obrigatório")
	}
	directoryMode := os.FileMode(0o750) | os.ModeSetgid
	if err := os.MkdirAll(clean, directoryMode); err != nil {
		return err
	}
	info, err := os.Lstat(clean)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("diretório do socket inválido")
	}
	uid, err := resolveID(owner, true)
	if err != nil {
		return err
	}
	gid, err := resolveID(group, false)
	if err != nil {
		return err
	}
	if uid >= 0 || gid >= 0 {
		if err = os.Chown(clean, uid, gid); err != nil {
			return fmt.Errorf("proprietário/grupo do diretório do socket: %w", err)
		}
	}
	return os.Chmod(clean, directoryMode)
}

func isSharedSystemDirectory(path string) bool {
	for _, shared := range []string{"/tmp", "/var", "/var/run", "/run"} {
		if path == shared {
			return true
		}
	}
	return false
}

func (o SocketOptions) normalized() (SocketOptions, error) {
	if o.Mode == 0 {
		o.Mode = 0o660
	}
	if o.Mode&^os.FileMode(0o777) != 0 || o.Mode&0o007 != 0 || o.Mode&0o600 != 0o600 {
		return o, errors.New("modo do socket deve ser restritivo e sem acesso para outros")
	}
	if !o.EnforcePeerCredentials {
		o.ExpectedPeerUID = -1
	}
	if o.ExpectedPeerUID < -1 || o.EnforcePeerCredentials && o.ExpectedPeerUID < 0 {
		return o, errors.New("UID esperado do peer inválido")
	}
	return o, nil
}

// Listen remains for compatibility with the first agent implementation.
// Runtime deployments should use ListenWithOptions to enforce a peer UID.
func Listen(socket string, _ http.Handler) (net.Listener, error) {
	return ListenWithOptions(socket, SocketOptions{ExpectedPeerUID: -1})
}

func ListenWithOptions(socket string, options SocketOptions) (net.Listener, error) {
	options, err := options.normalized()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(socket) == "" || !filepath.IsAbs(socket) && !strings.HasPrefix(socket, "./") && !strings.HasPrefix(socket, "../") {
		return nil, errors.New("caminho do socket inválido")
	}
	if options.ExpectedPeerUID >= 0 && !peerCredentialsAvailable() {
		return nil, errors.New("verificação de credencial do peer não é suportada nesta compilação")
	}
	if err = removeExistingSocket(socket); err != nil {
		return nil, err
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(socket, options.Mode); err != nil {
		_ = listener.Close()
		return nil, err
	}
	uid, err := resolveID(options.Owner, true)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	gid, err := resolveID(options.Group, false)
	if err != nil {
		_ = listener.Close()
		return nil, err
	}
	if uid >= 0 || gid >= 0 {
		if err = os.Chown(socket, uid, gid); err != nil {
			_ = listener.Close()
			return nil, fmt.Errorf("proprietário/grupo do socket: %w", err)
		}
	}
	return &credentialListener{Listener: listener, expectedUID: options.ExpectedPeerUID}, nil
}

func removeExistingSocket(socket string) error {
	info, err := os.Lstat(socket)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return errors.New("recusa remover caminho existente que não é socket")
	}
	return os.Remove(socket)
}

func resolveID(value string, isUser bool) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return -1, nil
	}
	if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
		return parsed, nil
	}
	if isUser {
		entry, err := user.Lookup(value)
		if err != nil {
			return -1, fmt.Errorf("usuário do socket %q: %w", value, err)
		}
		parsed, err := strconv.Atoi(entry.Uid)
		if err != nil || parsed < 0 {
			return -1, fmt.Errorf("UID do usuário do socket %q inválido", value)
		}
		return parsed, nil
	}
	entry, err := user.LookupGroup(value)
	if err != nil {
		return -1, fmt.Errorf("grupo do socket %q: %w", value, err)
	}
	parsed, err := strconv.Atoi(entry.Gid)
	if err != nil || parsed < 0 {
		return -1, fmt.Errorf("GID do grupo do socket %q inválido", value)
	}
	return parsed, nil
}

type credentialListener struct {
	net.Listener
	expectedUID int
}

func (l *credentialListener) Accept() (net.Conn, error) {
	for {
		conn, err := l.Listener.Accept()
		if err != nil || l.expectedUID < 0 {
			return conn, err
		}
		credentials, credentialErr := peerCredentials(conn)
		if credentialErr == nil && credentials.UID == l.expectedUID {
			return conn, nil
		}
		_ = conn.Close()
		// A rejected local peer must not terminate http.Server.Serve. A small
		// delay also prevents a hostile local process from spinning the accept loop.
		time.Sleep(10 * time.Millisecond)
	}
}

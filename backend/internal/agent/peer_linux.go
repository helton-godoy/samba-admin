//go:build linux

package agent

import (
	"fmt"
	"net"
	"syscall"
)

func peerCredentialsAvailable() bool { return true }

func peerCredentials(conn net.Conn) (PeerCredentials, error) {
	unixConn, ok := conn.(*net.UnixConn)
	if !ok {
		return PeerCredentials{}, ErrPeerCredentialUnsupported
	}
	raw, err := unixConn.SyscallConn()
	if err != nil {
		return PeerCredentials{}, err
	}
	var credentials *syscall.Ucred
	var controlErr error
	err = raw.Control(func(fd uintptr) {
		credentials, controlErr = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	if err != nil {
		return PeerCredentials{}, err
	}
	if controlErr != nil {
		return PeerCredentials{}, fmt.Errorf("SO_PEERCRED: %w", controlErr)
	}
	if credentials == nil {
		return PeerCredentials{}, ErrPeerCredentialUnsupported
	}
	return PeerCredentials{UID: int(credentials.Uid), GID: int(credentials.Gid)}, nil
}

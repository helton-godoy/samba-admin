//go:build !linux && !freebsd

package agent

import "net"

func peerCredentialsAvailable() bool { return false }

func peerCredentials(net.Conn) (PeerCredentials, error) {
	return PeerCredentials{}, ErrPeerCredentialUnsupported
}

package agent

import "errors"

var ErrPeerCredentialUnsupported = errors.New("credencial de peer não suportada")

type PeerCredentials struct {
	UID int
	GID int
}

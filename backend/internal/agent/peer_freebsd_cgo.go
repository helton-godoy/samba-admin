//go:build freebsd && cgo

package agent

/*
#include <sys/types.h>
#include <unistd.h>
#include <errno.h>

static int samba_admin_getpeereid(int fd, unsigned int *uid, unsigned int *gid) {
	uid_t peer_uid;
	gid_t peer_gid;
	if (getpeereid(fd, &peer_uid, &peer_gid) != 0) {
		return errno;
	}
	*uid = (unsigned int)peer_uid;
	*gid = (unsigned int)peer_gid;
	return 0;
}
*/
import "C"

import (
	"fmt"
	"net"
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
	var uid, gid C.uint
	var code C.int
	err = raw.Control(func(fd uintptr) {
		code = C.samba_admin_getpeereid(C.int(fd), &uid, &gid)
	})
	if err != nil {
		return PeerCredentials{}, err
	}
	if code != 0 {
		return PeerCredentials{}, fmt.Errorf("getpeereid falhou: errno %d", int(code))
	}
	return PeerCredentials{UID: int(uid), GID: int(gid)}, nil
}

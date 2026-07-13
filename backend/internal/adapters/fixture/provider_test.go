package fixture

import (
	"context"
	"testing"

	"github.com/hu-ufcat/samba-admin-backend/internal/fixturecollector"
)

func TestReplayUsesOnlyCapturedCommandVector(t *testing.T) {
	provider := New(fixturecollector.Fixture{Commands: []fixturecollector.CommandCapture{
		{Executable: "/sbin/mount", Arguments: []string{"-p"}, Stdout: "/dev/nda1p2 /srv/dados ufs rw,nfsv4acls 2 2\n"},
		{Executable: "/bin/df", Arguments: []string{"-k"}, Stdout: "Filesystem 1024-blocks Used Available Capacity Mounted on\n/dev/nda1p2 1048576 524288 524288 50% /srv/dados\n"},
		{Executable: "/bin/cat", Arguments: []string{"/etc/fstab"}, Stdout: "/dev/nda1p2 /srv/dados ufs rw,nfsv4acls 2 2\n"},
	}})
	filesystems, err := provider.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(filesystems) != 1 || filesystems[0].ACLModel != "nfsv4" {
		t.Fatalf("replay inesperado: %#v", filesystems)
	}
}

func TestReplayFailsClosedWhenCommandIsMissing(t *testing.T) {
	provider := New(fixturecollector.Fixture{})
	if _, err := provider.Inspect(context.Background()); err == nil {
		t.Fatal("fixture incompleta deveria falhar sem executar comando real")
	}
}

func TestReplayRealSanitizedFreeBSDFixture(t *testing.T) {
	provider, err := Open("../../../testdata/fixtures/freebsd-15.1-samba423-sanitized.json")
	if err != nil {
		t.Fatal(err)
	}
	system, err := provider.Inspect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if system.FreeBSDVersion != "15.1-RELEASE-p1" || system.Hostname != "{HOSTNAME}" {
		t.Fatalf("fixture real não foi sanitizada ou reproduzida: %+v", system)
	}
	samba, err := provider.Samba(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if samba.Package != "samba423-4.23.8_1" || samba.Version != "4.23.8" {
		t.Fatalf("inventário Samba divergente: %+v", samba)
	}
}

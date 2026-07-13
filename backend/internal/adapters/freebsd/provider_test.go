package freebsd

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fixtureRunner struct {
	outputs map[string]string
	seen    []string
}

func (r *fixtureRunner) Run(_ context.Context, path string, args ...string) (CommandResult, error) {
	key := path + " " + strings.Join(args, " ")
	r.seen = append(r.seen, key)
	value, ok := r.outputs[key]
	if !ok {
		return CommandResult{}, errors.New("fixture ausente: " + key)
	}
	return CommandResult{Stdout: value}, nil
}

func TestReadOnlyRunnerRejectsShellAndUnexpectedArguments(t *testing.T) {
	runner := ExecRunner{}
	if _, err := runner.Run(context.Background(), "/bin/sh", "-c", "id"); !errors.Is(err, ErrCommandDenied) {
		t.Fatalf("shell deveria ser recusado: %v", err)
	}
	if _, err := runner.Run(context.Background(), "/usr/local/bin/testparm", "-s", "/tmp/entrada"); !errors.Is(err, ErrCommandDenied) {
		t.Fatalf("argumento inesperado deveria ser recusado: %v", err)
	}
}

func TestFilesystemInventoryParsesUFSOptions(t *testing.T) {
	runner := &fixtureRunner{outputs: map[string]string{
		"/sbin/mount -p":      "/dev/nda1p1 / ufs rw 1 1\n/dev/nda1p2 /srv/dados ufs rw,nfsv4acls,userquota,groupquota 2 2\n",
		"/bin/df -k":          "Filesystem 1024-blocks Used Avail Capacity Mounted on\n/dev/nda1p1 1048576 524288 524288 50% /\n/dev/nda1p2 2097152 1048576 1048576 50% /srv/dados\n",
		"/bin/cat /etc/fstab": "/dev/nda1p1 / ufs rw 1 1\n/dev/nda1p2 /srv/dados ufs rw,nfsv4acls,userquota,groupquota 2 2\n",
	}}
	fileSystems, err := New(runner).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(fileSystems) != 2 {
		t.Fatalf("esperado dois sistemas de arquivos, obtido %d", len(fileSystems))
	}
	data := fileSystems[1]
	if data.ACLModel != "nfsv4" || !data.UserQuota || !data.GroupQuota || !data.FSTABPersistent {
		t.Fatalf("inventário UFS incorreto: %+v", data)
	}
}

func TestDomainDiagnosticNeverRequestsJoin(t *testing.T) {
	runner := &fixtureRunner{outputs: map[string]string{
		"/usr/local/bin/testparm -s":      "Server role: ROLE_DOMAIN_MEMBER\n[global]\n  workgroup = EBSERHNET\n  realm = EBSERHNET.EBSERH.GOV.BR\n  security = ADS\n  idmap config EBSERHNET : backend = rid\n  idmap config EBSERHNET : range = 2000000-2999999\n",
		"/bin/cat /etc/resolv.conf":       "nameserver 192.0.2.53\n",
		"/usr/local/bin/wbinfo --ping-dc": "checking the NETLOGON for domain[EBSERHNET] dc connection to \"dc.example\" succeeded\n",
	}}
	state, err := New(runner).Domain(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !state.Joined || state.IDMapStrategy != "rid" || state.Realm != "EBSERHNET.EBSERH.GOV.BR" {
		t.Fatalf("diagnóstico incorreto: %+v", state)
	}
	for _, call := range runner.seen {
		if strings.Contains(call, " join") || strings.Contains(call, " leave") || strings.Contains(call, " ads ") {
			t.Fatalf("adapter de leitura tentou operação mutável: %s", call)
		}
	}
}

func TestSambaConfigRedactsSecrets(t *testing.T) {
	value := redactSambaConfig("[global]\n  ldap admin dn = cn=admin\n  password = secret-value\n")
	if strings.Contains(value, "secret-value") {
		t.Fatalf("segredo não foi mascarado: %q", value)
	}
}

func TestSambaInventoryCapturesPackageVersion(t *testing.T) {
	runner := &fixtureRunner{outputs: map[string]string{
		"/usr/local/sbin/smbd -V":                  "Version 4.23.8",
		"/usr/local/sbin/pkg query %n-%v samba423": "samba423-4.23.8_1",
		"/usr/local/sbin/pkg info samba423":        "Name: samba423\nOptions: FULL_AUDIT: on\n",
		"/usr/local/bin/testparm -s":               "Server role: ROLE_STANDALONE\n",
		"/usr/local/bin/smbstatus --shares":        "",
		"/usr/local/bin/smbstatus --locks":         "",
	}}
	info, err := New(runner).Samba(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Package != "samba423-4.23.8_1" || info.Version != "4.23.8" {
		t.Fatalf("versão de Samba/pacote incorreta: %+v", info)
	}
	if !allowedReadOnlyInvocation("/usr/local/sbin/pkg", []string{"query", "%n-%v", "samba423"}) {
		t.Fatal("consulta fixa de versão do package foi recusada")
	}
}

func TestInventoryFailsClosedWithoutMinimumProbe(t *testing.T) {
	provider := New(&fixtureRunner{outputs: map[string]string{}})
	if _, err := provider.Samba(context.Background()); err == nil || !strings.Contains(err.Error(), "sondagem mínima do Samba") {
		t.Fatalf("Samba sem smbd -V deveria falhar de forma fechada: %v", err)
	}
	if _, err := provider.Cups(context.Background()); err == nil || !strings.Contains(err.Error(), "sondagem mínima do CUPS") {
		t.Fatalf("CUPS sem cupsd -t deveria falhar de forma fechada: %v", err)
	}
}

func TestCupsInventoryKeepsOptionalProbeFailuresStructured(t *testing.T) {
	runner := &fixtureRunner{outputs: map[string]string{
		"/usr/local/sbin/pkg query %n-%v cups": "cups-2.4.19_1",
		"/usr/local/sbin/cupsd -t":             "configuração válida",
	}}
	info, err := New(runner).Cups(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.Service != "valid" || info.Printers == nil || info.Backends == nil || len(info.Errors) != 2 {
		t.Fatalf("inventário CUPS parcial não foi estruturado: %+v", info)
	}
}

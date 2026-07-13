package validation

import "testing"

func TestSharePathPolicy(t *testing.T) {
	allowed := []string{"/srv/dados/setor", "/srv/dados/setor/sub", "/var/lib/samba/printers"}
	for _, p := range allowed {
		if err := SharePath(p); err != nil {
			t.Errorf("%s deveria ser permitido: %v", p, err)
		}
	}
	blocked := []string{"/root", "/etc", "/srv/dados/../../root", "/srv/dados/setor/..", "relative/path", "/usr/sbin/service", "/boot/kernel"}
	for _, p := range blocked {
		if err := SharePath(p); err == nil {
			t.Errorf("%s deveria ser bloqueado", p)
		}
	}
}

func TestPrincipalRejectsConfigurationInjection(t *testing.T) {
	for _, value := range []string{"DOMINIO\\grupo\nadmin users = invasor", "DOMINIO\\grupo;admin", "DOMINIO\\grupo#comentario"} {
		if err := Principal(value); err == nil {
			t.Fatalf("principal inseguro aceito: %q", value)
		}
	}
	if err := Principal("EBSERHNET\\Domain Users"); err != nil {
		t.Fatalf("principal AD válido recusado: %v", err)
	}
}

func TestConfigAllowlist(t *testing.T) {
	if err := ConfigPath("/usr/local/etc/smb4.conf"); err != nil {
		t.Fatal(err)
	}
	if err := ConfigPath("/etc/master.passwd"); err == nil {
		t.Fatal("arquivo sensível aceito")
	}
	if err := ConfigPath("/etc/../etc/fstab"); err == nil {
		t.Fatal("travessia de caminho de configuração aceita")
	}
}

package rbac

import "testing"

func TestActionResourceAuthorization(t *testing.T) {
	e := New()
	if !e.Allowed([]string{"administrador Samba"}, "write", "share") {
		t.Fatal("administrador Samba deveria editar share")
	}
	if e.Allowed([]string{"auditor"}, "write", "share") {
		t.Fatal("auditor não deve editar share")
	}
	if !e.Allowed([]string{"operador de impressão"}, "write", "printer") {
		t.Fatal("operador de impressão deveria editar impressora")
	}
	if e.Allowed([]string{"operador de arquivos"}, "rollback", "job") {
		t.Fatal("operador de arquivos não deve executar rollback genérico")
	}
	if !e.Allowed([]string{"administrador do sistema"}, "rollback", "configuration") {
		t.Fatal("administrador do sistema deveria poder iniciar rollback controlado")
	}
}

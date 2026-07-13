package fixturecollector

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/hu-ufcat/samba-admin-backend/internal/adapters/freebsd"
	"github.com/hu-ufcat/samba-admin-backend/internal/models"
)

type fixedRunner struct {
	result freebsd.CommandResult
	err    error
}

func (r fixedRunner) Run(context.Context, string, ...string) (freebsd.CommandResult, error) {
	return r.result, r.err
}

func TestCaptureRunnerRecordsOnlyDelegateResult(t *testing.T) {
	runner := NewCaptureRunner(fixedRunner{result: freebsd.CommandResult{Stdout: "ok", ExitCode: 0}})
	if _, err := runner.Run(context.Background(), "/bin/hostname", "-f"); err != nil {
		t.Fatal(err)
	}
	commands := runner.Commands()
	if len(commands) != 1 || commands[0].Executable != "/bin/hostname" || commands[0].Arguments[0] != "-f" {
		t.Fatalf("captura inesperada: %#v", commands)
	}
}

func TestSanitizeRemovesIdentifiersAndKeepsReplayVector(t *testing.T) {
	input := Fixture{
		SchemaVersion: SchemaVersion,
		Source:        "freebsd-readonly",
		System:        models.SystemInfo{Hostname: "nas", FQDN: "nas.hospital.gov.br"},
		Commands: []CommandCapture{{
			Executable: "/usr/local/bin/testparm",
			Arguments:  []string{"-s"},
			Stdout:     "host=nas.hospital.gov.br ip=192.168.122.180 user=DOMINIO\\operador sid=S-1-5-21-100-200-300-400 path=/srv/dados/paciente token=segredo",
		}},
	}
	result, err := Sanitize(input, SafeSanitizeOptions())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(result)
	text := string(raw)
	for _, forbidden := range []string{"nas.hospital.gov.br", "192.168.122.180", `DOMINIO\\operador`, "S-1-5-21-100-200-300-400", "/srv/dados/paciente", "segredo"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("identificador não sanitizado: %s em %s", forbidden, text)
		}
	}
	if result.Commands[0].Executable != "/usr/local/bin/testparm" || result.Commands[0].Arguments[0] != "-s" {
		t.Fatalf("vetor de replay foi alterado: %#v", result.Commands[0])
	}
}

func TestSanitizeRejectsPrivateKeyMaterial(t *testing.T) {
	_, err := Sanitize(Fixture{Commands: []CommandCapture{{Stdout: "-----BEGIN PRIVATE KEY-----"}}}, SafeSanitizeOptions())
	if err == nil {
		t.Fatal("material de chave privada deveria interromper a coleta")
	}
}

func TestSanitizePreservesContractIdentifiersAndFixedPaths(t *testing.T) {
	input := Fixture{
		Samba: models.SambaInfo{ConfigurationPath: "/usr/local/etc/smb4.conf"},
		Capabilities: []models.Capability{{
			ID: "domain.member.diagnose", Feature: "Diagnóstico", State: "nao_verificado", Scope: "FreeBSD/UFS",
		}},
	}
	result, err := Sanitize(input, SafeSanitizeOptions())
	if err != nil {
		t.Fatal(err)
	}
	if result.Samba.ConfigurationPath != "/usr/local/etc/smb4.conf" {
		t.Fatalf("caminho fixo do contrato foi alterado: %q", result.Samba.ConfigurationPath)
	}
	if result.Capabilities[0].ID != "domain.member.diagnose" || result.Capabilities[0].Scope != "FreeBSD/UFS" {
		t.Fatalf("metadado estrutural foi alterado: %#v", result.Capabilities[0])
	}
}

func TestCaptureRunnerKeepsReadError(t *testing.T) {
	runner := NewCaptureRunner(fixedRunner{err: errors.New("falha de leitura")})
	_, _ = runner.Run(context.Background(), "/bin/hostname")
	if got := runner.Commands()[0].Error; got != "falha de leitura" {
		t.Fatalf("erro não capturado: %q", got)
	}
}

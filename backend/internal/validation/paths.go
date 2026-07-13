package validation

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

var allowedShareRoots = []string{"/srv/dados", "/var/lib/samba/printers"}
var blockedSystemRoots = []string{"/boot", "/dev", "/proc", "/root", "/sbin", "/bin", "/lib", "/libexec", "/usr/bin", "/usr/sbin"}
var allowedConfigFiles = map[string]bool{"/usr/local/etc/smb4.conf": true, "/etc/rc.conf": true, "/etc/fstab": true, "/etc/krb5.conf": true, "/etc/nsswitch.conf": true, "/etc/syslog.conf": true, "/usr/local/etc/cups/cupsd.conf": true}

func SharePath(path string) error {
	if path == "" {
		return errors.New("caminho obrigatório")
	}
	if strings.ContainsAny(path, "\x00\r\n") {
		return errors.New("caminho contém caractere de controle")
	}
	clean := filepath.Clean(path)
	if clean != path {
		return errors.New("caminho deve estar em forma canônica e sem travessia")
	}
	if !filepath.IsAbs(clean) {
		return errors.New("caminho deve ser absoluto")
	}
	for _, b := range blockedSystemRoots {
		if clean == b || strings.HasPrefix(clean, b+string(filepath.Separator)) {
			return errors.New("diretório sensível bloqueado")
		}
	}
	for _, a := range allowedShareRoots {
		if clean == a || strings.HasPrefix(clean, a+string(filepath.Separator)) {
			return nil
		}
	}
	return errors.New("caminho fora dos volumes autorizados")
}
func ConfigPath(path string) error {
	if path == "" || strings.ContainsAny(path, "\x00\r\n") || path != filepath.Clean(path) || !filepath.IsAbs(path) {
		return errors.New("arquivo deve usar caminho absoluto e canônico")
	}
	if !allowedConfigFiles[path] {
		return errors.New("arquivo fora da lista autorizada")
	}
	return nil
}

func SingleLine(field, value string, maxLength int, allowEmpty bool) error {
	if !allowEmpty && strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s é obrigatório", field)
	}
	if maxLength > 0 && len(value) > maxLength {
		return fmt.Errorf("%s excede o tamanho permitido", field)
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return fmt.Errorf("%s deve ocupar uma única linha", field)
	}
	return nil
}

func Principal(value string) error {
	if err := SingleLine("principal", value, 256, false); err != nil {
		return err
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || strings.ContainsRune(`._$@ -\\`, character) {
			continue
		}
		return errors.New("principal contém caractere não permitido")
	}
	return nil
}
func ShareName(name string) error {
	if name == "" || len(name) > 80 {
		return errors.New("nome inválido")
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._$-", r)) {
			return errors.New("nome contém caractere não permitido")
		}
	}
	return nil
}

func IdempotencyKey(value string) error {
	if len(value) < 8 || len(value) > 128 {
		return fmt.Errorf("a chave de idempotência deve conter entre 8 e 128 caracteres")
	}
	for _, r := range value {
		if !(r == '-' || r == '_' || r == '.' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z') {
			return fmt.Errorf("a chave de idempotência contém caractere não permitido")
		}
	}
	return nil
}

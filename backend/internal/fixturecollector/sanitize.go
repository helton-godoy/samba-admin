package fixturecollector

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

type SanitizeOptions struct {
	Hostname bool
	IPs      bool
	Domains  bool
	Users    bool
	SIDs     bool
	Paths    bool
}

func SafeSanitizeOptions() SanitizeOptions {
	return SanitizeOptions{Hostname: true, IPs: true, Domains: true, Users: true, SIDs: true, Paths: true}
}

var (
	ipv4Pattern       = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	ipv6Pattern       = regexp.MustCompile(`(?i)\b(?:[0-9a-f]{1,4}:){2,7}[0-9a-f]{1,4}\b`)
	domainPattern     = regexp.MustCompile(`(?i)\b(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,63}\b`)
	sidPattern        = regexp.MustCompile(`(?i)\bS-1-(?:\d+-){1,14}\d+\b`)
	domainUserPattern = regexp.MustCompile(`\b[A-Za-z0-9_.-]+\\[A-Za-z0-9_.@ -]+\b`)
	pathPattern       = regexp.MustCompile(`(?:^|[[:space:]=:"'(])/(?:[A-Za-z0-9._@%+,-]+/)*[A-Za-z0-9._@%+,-]+`)
	secretAssignment  = regexp.MustCompile(`(?i)((?:password|passwd|secret|token|credential|authorization|cookie|private[_ -]?key|keytab|totp|otp|recovery[_ -]?code)[^\r\n=:]*[=:])[^\r\n]*`)
	privateKeyPattern = regexp.MustCompile(`(?i)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`)
	ticketPattern     = regexp.MustCompile(`(?i)(ticket cache:|krb5ccname=)`)
)

func Sanitize(input Fixture, options SanitizeOptions) (Fixture, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return Fixture{}, fmt.Errorf("serializar fixture: %w", err)
	}
	if privateKeyPattern.Match(raw) || ticketPattern.Match(raw) {
		return Fixture{}, errors.New("a coleta contém material proibido de chave privada ou cache Kerberos")
	}

	var document any
	if err = json.Unmarshal(raw, &document); err != nil {
		return Fixture{}, fmt.Errorf("decodificar fixture: %w", err)
	}
	redactor := newRedactor(input, options)
	document = redactValue(document, redactor)
	sanitizedJSON, err := json.Marshal(document)
	if err != nil {
		return Fixture{}, fmt.Errorf("serializar fixture sanitizada: %w", err)
	}
	var result Fixture
	if err = json.Unmarshal(sanitizedJSON, &result); err != nil {
		return Fixture{}, fmt.Errorf("validar fixture sanitizada: %w", err)
	}
	for i := range result.Commands {
		if i >= len(input.Commands) {
			break
		}
		result.Commands[i].Executable = input.Commands[i].Executable
		result.Commands[i].Arguments = append([]string(nil), input.Commands[i].Arguments...)
	}
	result.Sanitized = true
	return result, nil
}

type redactor struct {
	options      SanitizeOptions
	hostnames    []string
	pathTokens   map[string]string
	pathSequence int
}

func newRedactor(input Fixture, options SanitizeOptions) *redactor {
	hostnames := []string{input.System.FQDN, input.System.Hostname}
	return &redactor{options: options, hostnames: hostnames, pathTokens: map[string]string{}}
}

func redactValue(value any, redactor *redactor) any {
	switch typed := value.(type) {
	case string:
		return redactor.string(typed)
	case []any:
		for i := range typed {
			typed[i] = redactValue(typed[i], redactor)
		}
		return typed
	case map[string]any:
		_, capabilityFeature := typed["feature"]
		_, capabilityState := typed["state"]
		for key, item := range typed {
			if sensitiveKey(key) {
				typed[key] = "[REDACTED]"
				continue
			}
			if key == "id" && capabilityFeature && capabilityState {
				// IDs de capability pertencem ao contrato e não identificam o host.
				continue
			}
			if key == "configurationPath" && item == "/usr/local/etc/smb4.conf" {
				// Caminho fixo do provider FreeBSD, necessário para replay fiel.
				continue
			}
			typed[key] = redactValue(item, redactor)
		}
		return typed
	default:
		return typed
	}
}

func (r *redactor) string(value string) string {
	result := secretAssignment.ReplaceAllString(value, `${1}[REDACTED]`)
	if r.options.Hostname {
		for _, hostname := range r.hostnames {
			if strings.TrimSpace(hostname) != "" {
				result = replaceFold(result, hostname, "{HOSTNAME}")
			}
		}
	}
	if r.options.SIDs {
		result = sidPattern.ReplaceAllString(result, "{SID}")
	}
	if r.options.Users {
		result = domainUserPattern.ReplaceAllString(result, "{DOMAIN_USER}")
	}
	if r.options.IPs {
		result = ipv4Pattern.ReplaceAllString(result, "{IP}")
		result = ipv6Pattern.ReplaceAllString(result, "{IPV6}")
	}
	if r.options.Domains {
		result = domainPattern.ReplaceAllString(result, "{DOMAIN}")
	}
	if r.options.Paths {
		result = pathPattern.ReplaceAllStringFunc(result, func(match string) string {
			if strings.HasPrefix(match, "/") {
				return r.path(match)
			}
			return match[:1] + r.path(match[1:])
		})
	}
	return result
}

func (r *redactor) path(value string) string {
	for _, prefix := range []string{"/bin/", "/sbin/", "/usr/", "/etc/", "/dev/", "/var/run/", "/var/log/"} {
		if strings.HasPrefix(value, prefix) {
			return value
		}
	}
	if token, ok := r.pathTokens[value]; ok {
		return token
	}
	r.pathSequence++
	token := fmt.Sprintf("{PATH_%d}", r.pathSequence)
	r.pathTokens[value] = token
	return token
}

func replaceFold(value, old, replacement string) string {
	pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(old))
	return pattern.ReplaceAllString(value, replacement)
}

func sensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(key, "_", ""), "-", ""))
	for _, marker := range []string{"password", "passwd", "secret", "token", "credential", "authorization", "cookie", "privatekey", "keytab", "ticket", "totp", "otp", "recoverycode"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

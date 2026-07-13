package rbac

import "strings"

type Engine struct{ grants map[string]map[string]bool }

func New() *Engine {
	e := &Engine{grants: map[string]map[string]bool{}}
	e.add("auditor", "read:*")
	e.add("operador de arquivos", "read:*")
	e.add("operador de arquivos", "write:share")
	e.add("operador de arquivos", "write:acl")
	e.add("operador de arquivos", "write:file")
	e.add("operador de arquivos", "operate:job")
	e.add("operador de arquivos", "write:change")
	e.add("operador de impressão", "read:*")
	e.add("operador de impressão", "write:printer")
	e.add("operador de impressão", "operate:job")
	e.add("administrador Samba", "read:*")
	e.add("administrador Samba", "write:share")
	e.add("administrador Samba", "write:acl")
	e.add("administrador Samba", "write:domain")
	e.add("administrador Samba", "operate:samba")
	e.add("administrador Samba", "operate:job")
	e.add("administrador Samba", "write:change")
	e.add("administrador Samba", "operate:change")
	e.add("administrador do sistema", "*")
	e.add("administrador de segurança", "read:*")
	e.add("administrador de segurança", "write:audit")
	e.add("administrador de segurança", "write:logging")
	e.add("administrador de segurança", "write:rbac")
	e.add("administrador de segurança", "read:change")
	e.add("administrador de segurança", "approve:change")
	return e
}
func (e *Engine) add(role, permission string) {
	if e.grants[role] == nil {
		e.grants[role] = map[string]bool{}
	}
	e.grants[role][permission] = true
}
func (e *Engine) Allowed(roles []string, action, resource string) bool {
	want := action + ":" + resource
	for _, role := range roles {
		g := e.grants[role]
		if g["*"] || g[want] || g[action+":*"] {
			return true
		}
		for p := range g {
			if strings.HasSuffix(p, "*") && strings.HasPrefix(want, strings.TrimSuffix(p, "*")) {
				return true
			}
		}
	}
	return false
}

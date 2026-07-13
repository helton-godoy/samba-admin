package freebsd

import (
	"regexp"
	"strconv"
	"strings"
)

type mountEntry struct {
	Device, MountPoint, Type, Options string
}

type diskUsage struct{ size, used float64 }

func parseMountP(output string) []mountEntry {
	entries := []mountEntry{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		entries = append(entries, mountEntry{Device: fields[0], MountPoint: unescapeMount(fields[1]), Type: fields[2], Options: fields[3]})
	}
	return entries
}

func parseDF(output string) map[string]diskUsage {
	result := map[string]diskUsage{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[0] == "Filesystem" {
			continue
		}
		size, sizeErr := strconv.ParseFloat(fields[1], 64)
		used, usedErr := strconv.ParseFloat(fields[2], 64)
		if sizeErr == nil && usedErr == nil {
			result[unescapeMount(fields[len(fields)-1])] = diskUsage{size: size, used: used}
		}
	}
	return result
}

func parseFSTAB(output string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 3 {
			result[unescapeMount(fields[1])] = trimmed
		}
	}
	return result
}

func splitOptions(value string) []string {
	if value == "" || value == "-" {
		return []string{}
	}
	return strings.Split(value, ",")
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func kibToGiB(value float64) float64 { return value / (1024 * 1024) }

func unescapeMount(value string) string {
	return strings.NewReplacer("\\040", " ", "\\011", "\t", "\\134", "\\").Replace(value)
}

func parseLoadAverage(output string) []float64 {
	clean := strings.NewReplacer("{", "", "}", "", ",", " ").Replace(output)
	values := []float64{}
	for _, field := range strings.Fields(clean) {
		if value, err := strconv.ParseFloat(field, 64); err == nil {
			values = append(values, value)
		}
	}
	if len(values) > 3 {
		return values[:3]
	}
	return values
}

func parseSambaProfile(config string) string {
	lower := strings.ToLower(config)
	if strings.Contains(lower, "server role: role_active_directory_dc") || strings.Contains(lower, "server role = active directory domain controller") {
		return "ad-dc"
	}
	if strings.Contains(lower, "server role: role_domain_member") || strings.Contains(lower, "security = ads") {
		return "domain-member"
	}
	return "standalone"
}

func parseSambaAssignments(config string) map[string]string {
	values := map[string]string{}
	for _, line := range strings.Split(config, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])
		if key == "realm" || key == "workgroup" {
			values[key] = value
		}
	}
	return values
}

func parseSambaShares(config string) []string {
	shares := []string{}
	for _, line := range strings.Split(config, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") && !strings.EqualFold(trimmed, "[global]") {
			shares = append(shares, strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]"))
		}
	}
	return shares
}

func parseVFSModules(config string) []string {
	modules := map[string]bool{}
	for _, line := range strings.Split(config, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(strings.ToLower(parts[0])) != "vfs objects" {
			continue
		}
		for _, module := range strings.Fields(parts[1]) {
			modules[module] = true
		}
	}
	values := make([]string, 0, len(modules))
	for module := range modules {
		values = append(values, module)
	}
	return values
}

func parseIDMapStrategy(config string) string {
	for _, line := range strings.Split(config, "\n") {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(lower, "idmap config") && strings.Contains(lower, "backend") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func parseIDMapRanges(config string) (string, string) {
	for _, line := range strings.Split(config, "\n") {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(lower, "idmap config") && strings.Contains(lower, "range") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				return value, value
			}
		}
	}
	return "", ""
}

func parsePackageOptions(output string) []string {
	options := []string{}
	inOptions := false
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "Options        :" {
			inOptions = true
			continue
		}
		if inOptions && (trimmed == "" || strings.HasSuffix(trimmed, ":")) {
			break
		}
		if inOptions && trimmed != "" {
			options = append(options, trimmed)
		}
	}
	return options
}

func redactSambaConfig(config string) string {
	secret := regexp.MustCompile(`(?i)^(\s*(?:password|passdb\s+backend|ldap\s+admin\s+dn|idmap\s+config.*password)\s*=).*?$`)
	lines := strings.Split(config, "\n")
	for i, line := range lines {
		if secret.MatchString(line) {
			parts := strings.SplitN(line, "=", 2)
			lines[i] = parts[0] + "= [REDACTED]"
		}
	}
	return strings.Join(lines, "\n")
}

func countDataLines(output string) int {
	count := 0
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "PID") || strings.HasPrefix(trimmed, "Service") || strings.HasPrefix(trimmed, "Locked") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		count++
	}
	return count
}

func parseLPStatPrinters(output string) []string {
	result := []string{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "printer" {
			result = append(result, fields[1])
		}
	}
	return result
}

func parseLPStatDevices(output string) []string {
	result := []string{}
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "device for ") {
			result = append(result, strings.TrimSpace(line))
		}
	}
	return result
}

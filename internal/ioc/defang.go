package ioc

import "strings"

// Defang rewrites a value so it cannot be clicked or resolved when an export
// is opened. Types without a network form are returned unchanged. Values that
// are already defanged are not detected, so defanging twice nests brackets.
func Defang(value string, iocType IOCType) string {
	switch iocType {
	case IOCTypeIPv4:
		return strings.ReplaceAll(value, ".", "[.]")
	case IOCTypeIPv6:
		return strings.ReplaceAll(value, ":", "[:]")
	case IOCTypeDomain:
		idx := strings.LastIndex(value, ".")
		if idx >= 0 {
			return value[:idx] + "[.]" + value[idx+1:]
		}
		return value
	case IOCTypeURL:
		v := strings.ReplaceAll(value, ".", "[.]")
		v = strings.Replace(v, "https://", "hxxps://", 1)
		v = strings.Replace(v, "http://", "hxxp://", 1)
		v = strings.Replace(v, "ftp://", "fxp://", 1)
		return v
	case IOCTypeEmail:
		at := strings.Index(value, "@")
		if at < 0 {
			return value
		}
		domain := strings.ReplaceAll(value[at+1:], ".", "[.]")
		return value[:at] + "[@]" + domain
	default:
		return value
	}
}

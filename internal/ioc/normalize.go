package ioc

import (
	"net/netip"
	"strings"
)

// MatchKeyVersion identifies the normalization rules MatchKey applies. Stored
// alongside each indicator so a future rule change can find and re-key rows
// built under older rules.
const MatchKeyVersion = 1

// MatchKey returns the normalized key used to group IOC occurrences into a
// case-wide indicator. The raw value is never modified; only the returned key
// is normalized. Unknown types fall back to the trimmed raw value.
func MatchKey(iocType IOCType, raw string) string {
	v := strings.TrimSpace(raw)

	switch iocType {
	case IOCTypeDomain:
		return strings.TrimSuffix(strings.ToLower(v), ".")
	case IOCTypeEmail, IOCTypeMD5, IOCTypeSHA1, IOCTypeSHA256, IOCTypeRegistryKey:
		return strings.ToLower(v)
	case IOCTypeCVE:
		return strings.ToUpper(v)
	case IOCTypeIPv6:
		return ipv6Key(v)
	case IOCTypeURL:
		return urlKey(v)
	case IOCTypeFilePath:
		if isWindowsPath(v) {
			return strings.ToLower(v)
		}
		return v
	default:
		// ipv4 keeps leading zeros as written. file is exact match because
		// splitting a merged key later is lossy, while merging later is not.
		return v
	}
}

// ipv6Key returns the RFC 5952 compressed form, or the input unchanged if it
// does not parse.
func ipv6Key(v string) string {
	addr, err := netip.ParseAddr(v)
	if err != nil {
		return v
	}
	return addr.String()
}

// urlKey lowercases the scheme and host:port only. It splits the string by
// hand instead of using net/url, because URL.String() can re-encode the path
// and change what the analyst actually recorded.
func urlKey(v string) string {
	sep := strings.Index(v, "://")
	if sep < 0 {
		return v
	}
	scheme := v[:sep]
	rest := v[sep+3:]

	end := strings.IndexAny(rest, "/?#")
	if end < 0 {
		end = len(rest)
	}
	authority := rest[:end]
	tail := rest[end:]

	// Userinfo is case sensitive; only the host and port after the last @
	// are lowercased.
	if at := strings.LastIndex(authority, "@"); at >= 0 {
		authority = authority[:at+1] + strings.ToLower(authority[at+1:])
	} else {
		authority = strings.ToLower(authority)
	}

	return strings.ToLower(scheme) + "://" + authority + tail
}

// isWindowsPath reports whether v starts with a drive letter root (C:\ or
// c:/) or a UNC prefix (\\).
func isWindowsPath(v string) bool {
	if strings.HasPrefix(v, `\\`) {
		return true
	}
	if len(v) < 3 {
		return false
	}
	c := v[0]
	isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	return isLetter && v[1] == ':' && (v[2] == '\\' || v[2] == '/')
}

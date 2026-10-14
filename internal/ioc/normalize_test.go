package ioc

import "testing"

func TestMatchKey(t *testing.T) {
	cases := []struct {
		desc    string
		iocType IOCType
		raw     string
		want    string
	}{
		// domain
		{"domain mixed case", IOCTypeDomain, "Evil.Example.COM", "evil.example.com"},
		{"domain trailing dot", IOCTypeDomain, "evil.example.com.", "evil.example.com"},
		{"domain strips only one trailing dot", IOCTypeDomain, "evil.example.com..", "evil.example.com."},

		// email
		{"email lowercased", IOCTypeEmail, "Attacker@Evil.Example.com", "attacker@evil.example.com"},

		// hashes
		{"md5 upper", IOCTypeMD5, "D41D8CD98F00B204E9800998ECF8427E", "d41d8cd98f00b204e9800998ecf8427e"},
		{"sha1 upper", IOCTypeSHA1, "DA39A3EE5E6B4B0D3255BFEF95601890AFD80709", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"sha256 upper", IOCTypeSHA256, "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"sha256 lower", IOCTypeSHA256, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},

		// cve
		{"cve uppercased", IOCTypeCVE, "cve-2024-12345", "CVE-2024-12345"},

		// ipv4
		{"ipv4 leading zeros kept", IOCTypeIPv4, "192.168.001.105", "192.168.001.105"},

		// ipv6
		{"ipv6 full form compressed", IOCTypeIPv6, "2001:0db8:0000:0000:0000:0000:0000:0001", "2001:db8::1"},
		{"ipv6 upper compressed", IOCTypeIPv6, "2001:DB8::1", "2001:db8::1"},
		{"ipv6 invalid falls back", IOCTypeIPv6, "2001:db8::zz", "2001:db8::zz"},

		// url
		{"url scheme and host lowered, rest kept", IOCTypeURL,
			"HTTPS://Evil.Example.COM/Payload.EXE?Id=AbC#Frag",
			"https://evil.example.com/Payload.EXE?Id=AbC#Frag"},
		{"url userinfo kept, host and port lowered", IOCTypeURL,
			"https://User:Pass@Evil.COM:8443/X",
			"https://User:Pass@evil.com:8443/X"},
		{"url percent encoding untouched", IOCTypeURL,
			"HTTP://Evil.COM/a%2Fb/%7euser/%7EOther",
			"http://evil.com/a%2Fb/%7euser/%7EOther"},
		{"url query directly after host", IOCTypeURL, "HTTP://Evil.COM?Q=A", "http://evil.com?Q=A"},
		{"url fragment directly after host", IOCTypeURL, "HTTP://Evil.COM#Top", "http://evil.com#Top"},
		{"url host only", IOCTypeURL, "HTTP://Evil.COM", "http://evil.com"},
		{"url @ in path is not userinfo", IOCTypeURL, "HTTP://Evil.COM/User@Host", "http://evil.com/User@Host"},
		{"url no scheme falls back", IOCTypeURL, "Evil.COM/Path", "Evil.COM/Path"},

		// file_path
		{"file_path windows backslash", IOCTypeFilePath, `C:\Windows\System32\Evil.DLL`, `c:\windows\system32\evil.dll`},
		{"file_path windows forward slash", IOCTypeFilePath, `c:/Users/Bob/Evil.EXE`, `c:/users/bob/evil.exe`},
		{"file_path UNC", IOCTypeFilePath, `\\Server\Share\File.txt`, `\\server\share\file.txt`},
		{"file_path unix unchanged", IOCTypeFilePath, "/usr/Bin/Python3", "/usr/Bin/Python3"},
		{"file_path drive without separator is unix", IOCTypeFilePath, "C:Relative", "C:Relative"},

		// registry_key
		{"registry_key lowered, alias not expanded", IOCTypeRegistryKey, `HKLM\SOFTWARE\Microsoft`, `hklm\software\microsoft`},

		// file
		{"file upper exact", IOCTypeFile, "NTUSER.DAT", "NTUSER.DAT"},
		{"file lower exact", IOCTypeFile, "ntuser.dat", "ntuser.dat"},

		// unknown
		{"unknown type trimmed raw", IOCType("bitcoin"), "  1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa  ", "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if got := MatchKey(tc.iocType, tc.raw); got != tc.want {
				t.Errorf("MatchKey(%q, %q) = %q, want %q", tc.iocType, tc.raw, got, tc.want)
			}
		})
	}
}

// Each type must trim surrounding whitespace before applying its own rule.
func TestMatchKey_TrimsWhitespaceForEveryType(t *testing.T) {
	cases := []struct {
		iocType IOCType
		raw     string
		want    string
	}{
		{IOCTypeIPv4, "10.0.0.1", "10.0.0.1"},
		{IOCTypeIPv6, "2001:DB8::1", "2001:db8::1"},
		{IOCTypeDomain, "Evil.COM.", "evil.com"},
		{IOCTypeURL, "HTTP://Evil.COM/X", "http://evil.com/X"},
		{IOCTypeEmail, "A@B.COM", "a@b.com"},
		{IOCTypeMD5, "D41D8CD98F00B204E9800998ECF8427E", "d41d8cd98f00b204e9800998ecf8427e"},
		{IOCTypeSHA1, "DA39A3EE5E6B4B0D3255BFEF95601890AFD80709", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{IOCTypeSHA256, "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{IOCTypeFilePath, `C:\Temp\X.EXE`, `c:\temp\x.exe`},
		{IOCTypeFile, "NTUSER.DAT", "NTUSER.DAT"},
		{IOCTypeRegistryKey, `HKCU\Run`, `hkcu\run`},
		{IOCTypeCVE, "cve-2021-44228", "CVE-2021-44228"},
	}

	for _, tc := range cases {
		t.Run(string(tc.iocType), func(t *testing.T) {
			raw := " \t" + tc.raw + " \n"
			if got := MatchKey(tc.iocType, raw); got != tc.want {
				t.Errorf("MatchKey(%q, %q) = %q, want %q", tc.iocType, raw, got, tc.want)
			}
		})
	}
}

func TestMatchKey_Equivalence(t *testing.T) {
	pairs := []struct {
		desc    string
		iocType IOCType
		a, b    string
		same    bool
	}{
		{"domain case and trailing dot", IOCTypeDomain, "Evil.Example.COM", "evil.example.com.", true},
		{"sha256 case", IOCTypeSHA256,
			"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
			"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", true},
		{"ipv6 full vs compressed", IOCTypeIPv6, "2001:0db8:0000:0000:0000:0000:0000:0001", "2001:DB8::1", true},
		{"file case is significant", IOCTypeFile, "NTUSER.DAT", "ntuser.dat", false},
	}

	for _, tc := range pairs {
		t.Run(tc.desc, func(t *testing.T) {
			ka, kb := MatchKey(tc.iocType, tc.a), MatchKey(tc.iocType, tc.b)
			if (ka == kb) != tc.same {
				t.Errorf("keys %q and %q: same=%v, want same=%v", ka, kb, ka == kb, tc.same)
			}
		})
	}
}

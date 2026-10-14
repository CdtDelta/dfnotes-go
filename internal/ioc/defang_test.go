package ioc

import "testing"

func TestDefang(t *testing.T) {
	cases := []struct {
		name  string
		typ   IOCType
		value string
		want  string
	}{
		{"ipv4", IOCTypeIPv4, "192.168.1.13", "192[.]168[.]1[.]13"},
		{"ipv4 already defanged nests", IOCTypeIPv4, "10[.]0.0.1", "10[[.]]0[.]0[.]1"},
		{"ipv6", IOCTypeIPv6, "fe80::1", "fe80[:][:]1"},
		{"domain last dot only", IOCTypeDomain, "evil.example.com", "evil.example[.]com"},
		{"domain no dot", IOCTypeDomain, "localhost", "localhost"},
		{"domain already defanged", IOCTypeDomain, "example[.]com", "example[[.]]com"},
		{"url https", IOCTypeURL, "https://evil.example.com/a.php", "hxxps://evil[.]example[.]com/a[.]php"},
		{"url http", IOCTypeURL, "http://1.2.3.4/x", "hxxp://1[.]2[.]3[.]4/x"},
		{"url ftp", IOCTypeURL, "ftp://files.example.org", "fxp://files[.]example[.]org"},
		{"url no scheme", IOCTypeURL, "example.com/path", "example[.]com/path"},
		{"url already defanged", IOCTypeURL, "hxxps://a[.]b", "hxxps://a[[.]]b"},
		{"email", IOCTypeEmail, "first.last@mail.example.com", "first.last[@]mail[.]example[.]com"},
		{"email no at", IOCTypeEmail, "not-an-email", "not-an-email"},
		{"md5", IOCTypeMD5, "d41d8cd98f00b204e9800998ecf8427e", "d41d8cd98f00b204e9800998ecf8427e"},
		{"sha1", IOCTypeSHA1, "da39a3ee5e6b4b0d3255bfef95601890afd80709", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"sha256", IOCTypeSHA256, "E3B0C442", "E3B0C442"},
		{"file", IOCTypeFile, "evil.exe", "evil.exe"},
		{"file_path", IOCTypeFilePath, `C:\Windows\evil.dll`, `C:\Windows\evil.dll`},
		{"registry_key", IOCTypeRegistryKey, `HKLM\Software\Run`, `HKLM\Software\Run`},
		{"cve", IOCTypeCVE, "CVE-2024-1234", "CVE-2024-1234"},
		{"unknown type", IOCType("other"), "a.b", "a.b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Defang(tc.value, tc.typ); got != tc.want {
				t.Errorf("Defang(%q, %s) = %q, want %q", tc.value, tc.typ, got, tc.want)
			}
		})
	}
}

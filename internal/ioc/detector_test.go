package ioc

import (
	"testing"
)

func TestDetectIOCs_WinPath(t *testing.T) {
	cases := []struct {
		input     string
		wantValue string
		desc      string
	}{
		{
			input:     `The following file was found: C:\Windows\System32\evil.dll on the host`,
			wantValue: `C:\Windows\System32\evil.dll`,
			desc:      "path inside sentence: capture only the path, not trailing text",
		},
		{
			input:     `dropped C:\Temp\payload.exe here`,
			wantValue: `C:\Temp\payload.exe`,
			desc:      "path at mid-sentence with no directory depth",
		},
		{
			input:     `C:\Windows\System32\evil.dll`,
			wantValue: `C:\Windows\System32\evil.dll`,
			desc:      "path alone (no trailing text)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			matches := DetectIOCs(tc.input)
			var paths []string
			for _, m := range matches {
				if m.Type == IOCTypeFilePath {
					paths = append(paths, m.Value)
				}
			}
			if len(paths) != 1 {
				t.Fatalf("input %q: expected 1 file path IOC, got %d: %v", tc.input, len(paths), paths)
			}
			if paths[0] != tc.wantValue {
				t.Errorf("input %q: got %q, want %q", tc.input, paths[0], tc.wantValue)
			}
		})
	}
}

func TestDetectIOCs_IPv4(t *testing.T) {
	cases := []struct {
		input   string
		wantHit bool
		desc    string
	}{
		{"connect to 192.168.1.1 now", true, "plain IPv4 should match"},
		{"ip is 10.0.0.1", true, "IPv4 at end of string should match"},
		{"addr: 255.255.255.0/24", true, "IPv4 followed by slash should match"},
		{"version 1.2.3.4.5 installed", false, "five-segment version string must not match"},
		{"agent/1.2.3.4.5 (build)", false, "five-segment version in user-agent must not match"},
		{"1.2.3.4.56 suffix", false, "fifth segment with two digits must not match"},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			matches := DetectIOCs(tc.input)
			found := false
			for _, m := range matches {
				if m.Type == IOCTypeIPv4 {
					found = true
					break
				}
			}
			if found != tc.wantHit {
				t.Errorf("input %q: got IPv4 match=%v, want %v", tc.input, found, tc.wantHit)
			}
		})
	}
}

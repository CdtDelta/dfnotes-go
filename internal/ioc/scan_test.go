package ioc

import (
	"reflect"
	"testing"
)

func candidate(id string, t IOCType, value string) Indicator {
	return Indicator{IndicatorID: id, Type: t, Value: value, MatchKey: MatchKey(t, value)}
}

func rawValues(matches []KnownMatch) []string {
	out := []string{}
	for _, m := range matches {
		out = append(out, m.RawValue)
	}
	return out
}

func TestFindKnownValuesFileMatches(t *testing.T) {
	ntuser := []Indicator{candidate("f1", IOCTypeFile, "NTUSER.DAT")}
	for _, content := range []string{
		`C:\Users\tom\NTUSER.DAT`,
		`the hive "NTUSER.DAT" was copied`,
		`NTUSER.DAT was copied`,
		`copied NTUSER.DAT`,
		`copied NTUSER.DAT, then parsed it`,
		`copied NTUSER.DAT. Then parsed it`,
		`/mnt/c/Users/tom/NTUSER.DAT`,
		`(NTUSER.DAT)`,
	} {
		got := FindKnownValues(content, ntuser)
		want := []KnownMatch{{IndicatorID: "f1", Type: IOCTypeFile, RawValue: "NTUSER.DAT"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("FindKnownValues(%q) = %+v, want %+v", content, got, want)
		}
	}
}

func TestFindKnownValuesFileRejects(t *testing.T) {
	ntuser := []Indicator{candidate("f1", IOCTypeFile, "NTUSER.DAT")}
	for _, content := range []string{
		"ntuser.dat",
		"NTUSER.DAT.LOG",
		"NTUSER.DAT1",
		"XNTUSER.DAT",
		"NTUSER.DAT_old",
		"old-NTUSER.DAT",
		"backup.NTUSER.DAT",
	} {
		if got := FindKnownValues(content, ntuser); len(got) != 0 {
			t.Errorf("FindKnownValues(%q) = %+v, want no match", content, got)
		}
	}
}

func TestFindKnownValuesDomain(t *testing.T) {
	evil := []Indicator{candidate("d1", IOCTypeDomain, "evil.com")}

	got := FindKnownValues("beacon to Evil.COM observed", evil)
	want := []KnownMatch{{IndicatorID: "d1", Type: IOCTypeDomain, RawValue: "Evil.COM"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("case-folded domain = %+v, want %+v", got, want)
	}

	for _, content := range []string{"sub.evil.com", "evil.com.au", "not-evil.com"} {
		if got := FindKnownValues(content, evil); len(got) != 0 {
			t.Errorf("FindKnownValues(%q) = %+v, want no match", content, got)
		}
	}
}

func TestFindKnownValuesURL(t *testing.T) {
	url := []Indicator{candidate("u1", IOCTypeURL, "https://x.com/A")}

	if got := FindKnownValues("fetched https://x.com/a today", url); len(got) != 0 {
		t.Errorf("path differing in case matched: %+v", got)
	}
	if got := FindKnownValues("fetched https://x.com/A/b today", url); len(got) != 0 {
		t.Errorf("longer path matched: %+v", got)
	}

	got := FindKnownValues("fetched HTTPS://X.COM/A today", url)
	if want := []string{"HTTPS://X.COM/A"}; !reflect.DeepEqual(rawValues(got), want) {
		t.Errorf("scheme and host case folding = %v, want %v", rawValues(got), want)
	}
}

func TestFindKnownValuesFilePathCaseFolding(t *testing.T) {
	win := []Indicator{candidate("p1", IOCTypeFilePath, `C:\Temp\evil.exe`)}
	got := FindKnownValues(`ran c:\temp\EVIL.EXE twice`, win)
	if want := []string{`c:\temp\EVIL.EXE`}; !reflect.DeepEqual(rawValues(got), want) {
		t.Errorf("windows path = %v, want %v", rawValues(got), want)
	}
	if got := FindKnownValues(`ran C:\Temp\evil.exe\x`, win); len(got) != 0 {
		t.Errorf("windows path followed by a separator matched: %+v", got)
	}

	unix := []Indicator{candidate("p2", IOCTypeFilePath, "/tmp/Evil")}
	if got := FindKnownValues("ran /tmp/evil", unix); len(got) != 0 {
		t.Errorf("unix path matched with different case: %+v", got)
	}
	if got := FindKnownValues("ran /tmp/Evil/sub", unix); len(got) != 0 {
		t.Errorf("unix path followed by a separator matched: %+v", got)
	}
}

func TestFindKnownValuesMetacharacters(t *testing.T) {
	c := []Indicator{candidate("f1", IOCTypeFile, "report(1)+final.txt")}
	got := FindKnownValues("opened report(1)+final.txt from email", c)
	if want := []string{"report(1)+final.txt"}; !reflect.DeepEqual(rawValues(got), want) {
		t.Errorf("metacharacter value = %v, want %v", rawValues(got), want)
	}
	if got := FindKnownValues("opened report1final.txt", c); len(got) != 0 {
		t.Errorf("value matched as a regex: %+v", got)
	}
}

func TestFindKnownValuesNonASCII(t *testing.T) {
	c := []Indicator{
		candidate("f1", IOCTypeFile, "NTUSER.DAT"),
		candidate("d1", IOCTypeDomain, "evil.com"),
	}
	got := FindKnownValues("Москва NTUSER.DAT café EVIL.com ß", c)
	want := []KnownMatch{
		{IndicatorID: "f1", Type: IOCTypeFile, RawValue: "NTUSER.DAT"},
		{IndicatorID: "d1", Type: IOCTypeDomain, RawValue: "EVIL.com"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("non-ASCII content = %+v, want %+v", got, want)
	}

	// A non-ASCII letter directly before the value is still a letter.
	if got := FindKnownValues("éNTUSER.DAT", c[:1]); len(got) != 0 {
		t.Errorf("value glued to a non-ASCII letter matched: %+v", got)
	}
}

func TestFindKnownValuesDedupe(t *testing.T) {
	c := []Indicator{candidate("f1", IOCTypeFile, "NTUSER.DAT")}
	got := FindKnownValues("NTUSER.DAT and NTUSER.DAT and again NTUSER.DAT", c)
	if len(got) != 1 {
		t.Errorf("got %d matches, want 1: %+v", len(got), got)
	}

	// Different raw spellings of one indicator are separate occurrences.
	d := []Indicator{candidate("d1", IOCTypeDomain, "evil.com")}
	got = FindKnownValues("evil.com then EVIL.COM then evil.com", d)
	if want := []string{"evil.com", "EVIL.COM"}; !reflect.DeepEqual(rawValues(got), want) {
		t.Errorf("raw values = %v, want %v", rawValues(got), want)
	}
}

func TestFindKnownValuesOrder(t *testing.T) {
	c := []Indicator{
		candidate("d1", IOCTypeDomain, "evil.com"),
		candidate("f1", IOCTypeFile, "NTUSER.DAT"),
	}
	got := FindKnownValues("NTUSER.DAT then evil.com then EVIL.COM", c)
	want := []KnownMatch{
		{IndicatorID: "d1", Type: IOCTypeDomain, RawValue: "evil.com"},
		{IndicatorID: "d1", Type: IOCTypeDomain, RawValue: "EVIL.COM"},
		{IndicatorID: "f1", Type: IOCTypeFile, RawValue: "NTUSER.DAT"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("order = %+v, want %+v", got, want)
	}
}

func TestFindKnownValuesEmpty(t *testing.T) {
	c := []Indicator{candidate("f1", IOCTypeFile, "NTUSER.DAT")}
	for name, got := range map[string][]KnownMatch{
		"nil candidates":   FindKnownValues("NTUSER.DAT", nil),
		"empty candidates": FindKnownValues("NTUSER.DAT", []Indicator{}),
		"empty content":    FindKnownValues("", c),
	} {
		if got == nil || len(got) != 0 {
			t.Errorf("%s = %#v, want empty non-nil slice", name, got)
		}
	}
}

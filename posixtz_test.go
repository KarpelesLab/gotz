package gotz

import (
	"testing"
)

func TestParsePosixTZSimple(t *testing.T) {
	p, err := ParsePosixTZ("EST5EDT,M3.2.0,M11.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if p.StdAbbrev != "EST" {
		t.Errorf("StdAbbrev = %q, want EST", p.StdAbbrev)
	}
	if p.StdOffset != -5*3600 {
		t.Errorf("StdOffset = %d, want %d", p.StdOffset, -5*3600)
	}
	if p.DSTAbbrev != "EDT" {
		t.Errorf("DSTAbbrev = %q, want EDT", p.DSTAbbrev)
	}
	if p.DSTOffset != -4*3600 {
		t.Errorf("DSTOffset = %d, want %d", p.DSTOffset, -4*3600)
	}
	if !p.HasDST() {
		t.Error("HasDST() should be true")
	}
	if p.Start.Kind != RuleMonthWeekDay || p.Start.Mon != 3 || p.Start.Week != 2 || p.Start.Day != 0 {
		t.Errorf("Start = %+v", p.Start)
	}
	if p.End.Kind != RuleMonthWeekDay || p.End.Mon != 11 || p.End.Week != 1 || p.End.Day != 0 {
		t.Errorf("End = %+v", p.End)
	}
}

func TestParsePosixTZNoDST(t *testing.T) {
	p, err := ParsePosixTZ("JST-9")
	if err != nil {
		t.Fatal(err)
	}
	if p.StdAbbrev != "JST" {
		t.Errorf("StdAbbrev = %q", p.StdAbbrev)
	}
	if p.StdOffset != 9*3600 {
		t.Errorf("StdOffset = %d, want %d", p.StdOffset, 9*3600)
	}
	if p.HasDST() {
		t.Error("HasDST() should be false")
	}
}

func TestParsePosixTZQuoted(t *testing.T) {
	p, err := ParsePosixTZ("<-05>5<-04>,M3.2.0,M11.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if p.StdAbbrev != "-05" {
		t.Errorf("StdAbbrev = %q, want -05", p.StdAbbrev)
	}
	if p.DSTAbbrev != "-04" {
		t.Errorf("DSTAbbrev = %q, want -04", p.DSTAbbrev)
	}
}

func TestParsePosixTZWithTime(t *testing.T) {
	p, err := ParsePosixTZ("CET-1CEST,M3.5.0/2,M10.5.0/3")
	if err != nil {
		t.Fatal(err)
	}
	if p.StdAbbrev != "CET" {
		t.Errorf("StdAbbrev = %q", p.StdAbbrev)
	}
	if p.StdOffset != 3600 {
		t.Errorf("StdOffset = %d", p.StdOffset)
	}
	if p.DSTAbbrev != "CEST" {
		t.Errorf("DSTAbbrev = %q", p.DSTAbbrev)
	}
	if p.Start.Time != 7200 {
		t.Errorf("Start.Time = %d, want 7200", p.Start.Time)
	}
	if p.End.Time != 10800 {
		t.Errorf("End.Time = %d, want 10800", p.End.Time)
	}
}

func TestPosixTZString(t *testing.T) {
	tests := []string{
		"EST5EDT,M3.2.0,M11.1.0",
		"JST-9",
		"CET-1CEST,M3.5.0,M10.5.0/3",
	}
	for _, s := range tests {
		p, err := ParsePosixTZ(s)
		if err != nil {
			t.Errorf("ParsePosixTZ(%q): %v", s, err)
			continue
		}
		got := p.String()
		if got != s {
			t.Errorf("String() = %q, want %q", got, s)
		}
	}
}

func TestPosixTZLookup(t *testing.T) {
	p, err := ParsePosixTZ("EST5EDT,M3.2.0,M11.1.0")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		unix   int64
		abbrev string
		offset int
		isDST  bool
	}{
		// 2024-01-15 12:00:00 UTC — EST
		{1705320000, "EST", -18000, false},
		// 2024-07-15 12:00:00 UTC — EDT
		{1721044800, "EDT", -14400, true},
	}
	for _, tt := range tests {
		abbrev, offset, isDST := p.Lookup(tt.unix)
		if abbrev != tt.abbrev || offset != tt.offset || isDST != tt.isDST {
			t.Errorf("Lookup(%d) = (%q, %d, %v), want (%q, %d, %v)",
				tt.unix, abbrev, offset, isDST, tt.abbrev, tt.offset, tt.isDST)
		}
	}
}

func TestPosixTZTransitionsForYear(t *testing.T) {
	p, err := ParsePosixTZ("EST5EDT,M3.2.0,M11.1.0")
	if err != nil {
		t.Fatal(err)
	}

	start, end, ok := p.TransitionsForYear(2024)
	if !ok {
		t.Fatal("TransitionsForYear returned ok=false")
	}

	// 2024 DST starts: March 10, 2024 07:00:00 UTC (2:00 AM EST)
	// 2024 DST ends: November 3, 2024 06:00:00 UTC (2:00 AM EDT)
	if start != 1710054000 {
		t.Errorf("DST start = %d, want 1710054000", start)
	}
	if end != 1730613600 {
		t.Errorf("DST end = %d, want 1730613600", end)
	}
}

func TestPosixTZNoDSTLookup(t *testing.T) {
	p, err := ParsePosixTZ("JST-9")
	if err != nil {
		t.Fatal(err)
	}

	abbrev, offset, isDST := p.Lookup(1705320000)
	if abbrev != "JST" || offset != 9*3600 || isDST {
		t.Errorf("Lookup = (%q, %d, %v)", abbrev, offset, isDST)
	}

	_, _, ok := p.TransitionsForYear(2024)
	if ok {
		t.Error("TransitionsForYear should return ok=false for no-DST zone")
	}
}

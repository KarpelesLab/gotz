package gotz

import (
	"testing"
	"time"
)

func TestLoadUTC(t *testing.T) {
	z, err := Load("UTC")
	if err != nil {
		t.Fatal(err)
	}
	if z.Name() != "UTC" {
		t.Errorf("Name() = %q, want %q", z.Name(), "UTC")
	}
	if len(z.Types()) != 1 {
		t.Fatalf("Types() len = %d, want 1", len(z.Types()))
	}
	zt := z.Types()[0]
	if zt.Offset != 0 || zt.IsDST || zt.Abbrev != "UTC" {
		t.Errorf("UTC zone type = %+v", zt)
	}
}

func TestLoadNewYork(t *testing.T) {
	z, err := Load("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	if z.Name() != "America/New_York" {
		t.Errorf("Name() = %q", z.Name())
	}
	if z.Version() < 2 {
		t.Errorf("Version() = %d, want >= 2", z.Version())
	}

	types := z.Types()
	if len(types) < 2 {
		t.Fatalf("Types() len = %d, want >= 2", len(types))
	}

	// Should have EST and EDT among types.
	foundEST, foundEDT := false, false
	for _, zt := range types {
		switch zt.Abbrev {
		case "EST":
			foundEST = true
			if zt.Offset != -5*3600 {
				t.Errorf("EST offset = %d, want %d", zt.Offset, -5*3600)
			}
			if zt.IsDST {
				t.Error("EST should not be DST")
			}
		case "EDT":
			foundEDT = true
			if zt.Offset != -4*3600 {
				t.Errorf("EDT offset = %d, want %d", zt.Offset, -4*3600)
			}
			if !zt.IsDST {
				t.Error("EDT should be DST")
			}
		}
	}
	if !foundEST {
		t.Error("did not find EST in types")
	}
	if !foundEDT {
		t.Error("did not find EDT in types")
	}

	trans := z.Transitions()
	if len(trans) < 100 {
		t.Errorf("Transitions() len = %d, want >= 100", len(trans))
	}

	if z.Extend() == nil {
		t.Error("Extend() is nil, want POSIX TZ rule")
	}
	if z.ExtendRaw() == "" {
		t.Error("ExtendRaw() is empty")
	}
}

func TestLoadTokyo(t *testing.T) {
	z, err := Load("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	// Tokyo has JST at +9 and no current DST.
	types := z.Types()
	foundJST := false
	for _, zt := range types {
		if zt.Abbrev == "JST" {
			foundJST = true
			if zt.Offset != 9*3600 {
				t.Errorf("JST offset = %d, want %d", zt.Offset, 9*3600)
			}
		}
	}
	if !foundJST {
		t.Error("did not find JST in types")
	}
}

func TestLookup(t *testing.T) {
	z, err := Load("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	// 2024-01-15 12:00:00 UTC — should be EST.
	winter := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	zt := z.Lookup(winter)
	if zt.Abbrev != "EST" {
		t.Errorf("winter lookup = %q, want EST", zt.Abbrev)
	}

	// 2024-07-15 12:00:00 UTC — should be EDT.
	summer := time.Date(2024, 7, 15, 12, 0, 0, 0, time.UTC)
	zt = z.Lookup(summer)
	if zt.Abbrev != "EDT" {
		t.Errorf("summer lookup = %q, want EDT", zt.Abbrev)
	}
}

func TestLocationRoundTrip(t *testing.T) {
	z, err := Load("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}

	loc, err := z.Location()
	if err != nil {
		t.Fatal(err)
	}

	// Compare with Go's own LoadLocation.
	goLoc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}

	// Check a specific time in both locations.
	testTime := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	name1, off1 := testTime.In(loc).Zone()
	name2, off2 := testTime.In(goLoc).Zone()

	if name1 != name2 || off1 != off2 {
		t.Errorf("Location zone = (%q, %d), Go = (%q, %d)", name1, off1, name2, off2)
	}
}

func TestTransitionsForRange(t *testing.T) {
	z, err := Load("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	// Get transitions for 2024 — should have 2 (DST start + end),
	// generated from the POSIX extend rule since stored transitions
	// typically stop around 2007.
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	trans := z.TransitionsForRange(start, end)

	if len(trans) != 2 {
		t.Fatalf("TransitionsForRange(2024) len = %d, want 2", len(trans))
	}

	types := z.Types()

	// First transition: std -> DST (March)
	t0 := trans[0]
	if types[t0.Type].Abbrev != "EDT" {
		t.Errorf("first transition type = %q, want EDT", types[t0.Type].Abbrev)
	}
	t0Time := time.Unix(t0.When, 0).UTC()
	if t0Time.Month() != time.March {
		t.Errorf("DST start month = %v, want March", t0Time.Month())
	}

	// Second transition: DST -> std (November)
	t1 := trans[1]
	if types[t1.Type].Abbrev != "EST" {
		t.Errorf("second transition type = %q, want EST", types[t1.Type].Abbrev)
	}
	t1Time := time.Unix(t1.When, 0).UTC()
	if t1Time.Month() != time.November {
		t.Errorf("DST end month = %v, want November", t1Time.Month())
	}
}

func TestTransitionsForRangeMultiYear(t *testing.T) {
	z, err := Load("America/New_York")
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	trans := z.TransitionsForRange(start, end)

	// 3 years * 2 transitions = 6
	if len(trans) != 6 {
		t.Fatalf("TransitionsForRange(2024-2026) len = %d, want 6", len(trans))
	}

	// Verify chronological order.
	for i := 1; i < len(trans); i++ {
		if trans[i].When <= trans[i-1].When {
			t.Errorf("transition %d (when=%d) not after %d (when=%d)",
				i, trans[i].When, i-1, trans[i-1].When)
		}
	}
}

func TestTransitionsForRangeNoDST(t *testing.T) {
	z, err := Load("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	trans := z.TransitionsForRange(start, end)

	if len(trans) != 0 {
		t.Errorf("TransitionsForRange for Asia/Tokyo (no DST) len = %d, want 0", len(trans))
	}
}

func TestLoadCached(t *testing.T) {
	z1, err := Load("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	z2, err := Load("Europe/London")
	if err != nil {
		t.Fatal(err)
	}
	if z1 != z2 {
		t.Error("Load should return the same *Zone for the same name")
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("Fake/Timezone")
	if err == nil {
		t.Error("expected error for non-existent timezone")
	}
}

func TestParse(t *testing.T) {
	// Load raw data from the zip and parse it directly.
	data, err := readFromZip("Europe/Paris")
	if err != nil {
		t.Fatal(err)
	}

	z, err := Parse("Europe/Paris", data)
	if err != nil {
		t.Fatal(err)
	}
	if z.Name() != "Europe/Paris" {
		t.Errorf("Name() = %q", z.Name())
	}

	foundCET := false
	for _, zt := range z.Types() {
		if zt.Abbrev == "CET" {
			foundCET = true
			if zt.Offset != 3600 {
				t.Errorf("CET offset = %d, want 3600", zt.Offset)
			}
		}
	}
	if !foundCET {
		t.Error("did not find CET in types")
	}
}

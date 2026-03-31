// Package gotz provides direct access to IANA timezone data, exposing
// transitions, zone types, and POSIX TZ rules that Go's time.Location
// keeps private.
//
// Timezone data is compiled from the official IANA source and embedded
// in the package. Use Load to get a Zone by IANA name:
//
//	z, err := gotz.Load("America/New_York")
//	for _, t := range z.Transitions() {
//	    fmt.Println(t.When, z.Types()[t.Type].Abbrev)
//	}
package gotz

import (
	"archive/zip"
	"bytes"
	_ "embed"
	"fmt"
	"sync"
	"time"
)

//go:embed zoneinfo.zip
var zoneinfoZip []byte

var (
	cache   sync.Map // map[string]*Zone
	zipOnce sync.Once
	zipR    *zip.Reader
	zipErr  error
)

func getZipReader() (*zip.Reader, error) {
	zipOnce.Do(func() {
		zipR, zipErr = zip.NewReader(bytes.NewReader(zoneinfoZip), int64(len(zoneinfoZip)))
	})
	return zipR, zipErr
}

// Zone represents a parsed IANA timezone with all raw data exposed.
type Zone struct {
	name        string
	version     int
	types       []ZoneType
	transitions []Transition
	leapSeconds []LeapSecond
	extend      *PosixTZ
	extendRaw   string
	rawData     []byte
}

// ZoneType describes a local time type (e.g., EST, EDT).
type ZoneType struct {
	Abbrev string // abbreviated name
	Offset int    // seconds east of UTC
	IsDST  bool   // true if daylight saving time
}

// Transition represents a moment when the timezone rule changes.
type Transition struct {
	When  int64 // Unix timestamp
	Type  int   // index into Zone.Types()
	IsStd bool  // transition time is standard (not wall clock)
	IsUT  bool  // transition time is UT (not local)
}

// LeapSecond represents a leap second record.
type LeapSecond struct {
	When       int64 // Unix timestamp
	Correction int32 // cumulative correction
}

// Load returns a Zone for the given IANA timezone name.
// Results are cached; subsequent calls for the same name return the same *Zone.
func Load(name string) (*Zone, error) {
	if name == "" || name == "UTC" {
		return loadUTC(), nil
	}

	if v, ok := cache.Load(name); ok {
		return v.(*Zone), nil
	}

	data, err := readFromZip(name)
	if err != nil {
		return nil, fmt.Errorf("gotz: zone %q: %w", name, err)
	}

	z, err := Parse(name, data)
	if err != nil {
		return nil, err
	}

	if actual, loaded := cache.LoadOrStore(name, z); loaded {
		return actual.(*Zone), nil
	}
	return z, nil
}

// Parse parses TZif-format binary data into a Zone.
func Parse(name string, data []byte) (*Zone, error) {
	return parseData(name, data)
}

func readFromZip(name string) ([]byte, error) {
	r, err := getZipReader()
	if err != nil {
		return nil, err
	}
	for _, f := range r.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			buf := make([]byte, f.UncompressedSize64)
			n, err := rc.Read(buf)
			if err != nil && err.Error() != "EOF" {
				return nil, err
			}
			return buf[:n], nil
		}
	}
	return nil, fmt.Errorf("not found in embedded data")
}

func loadUTC() *Zone {
	if v, ok := cache.Load("UTC"); ok {
		return v.(*Zone)
	}
	z := &Zone{
		name:    "UTC",
		version: 2,
		types:   []ZoneType{{Abbrev: "UTC", Offset: 0, IsDST: false}},
	}
	if actual, loaded := cache.LoadOrStore("UTC", z); loaded {
		return actual.(*Zone)
	}
	return z
}

// Name returns the IANA timezone name.
func (z *Zone) Name() string { return z.name }

// Version returns the TZif format version (1, 2, 3, or 4).
func (z *Zone) Version() int { return z.version }

// Types returns a copy of the zone type definitions.
func (z *Zone) Types() []ZoneType {
	out := make([]ZoneType, len(z.types))
	copy(out, z.types)
	return out
}

// Transitions returns a copy of the transition records.
func (z *Zone) Transitions() []Transition {
	out := make([]Transition, len(z.transitions))
	copy(out, z.transitions)
	return out
}

// LeapSeconds returns a copy of the leap second records.
func (z *Zone) LeapSeconds() []LeapSecond {
	out := make([]LeapSecond, len(z.leapSeconds))
	copy(out, z.leapSeconds)
	return out
}

// Extend returns the parsed POSIX TZ rule for computing future transitions,
// or nil if the TZif file has no footer string.
func (z *Zone) Extend() *PosixTZ { return z.extend }

// ExtendRaw returns the raw POSIX TZ footer string.
func (z *Zone) ExtendRaw() string { return z.extendRaw }

// String returns the timezone name.
func (z *Zone) String() string { return z.name }

// Lookup returns the zone type in effect at the given time.
// It searches transitions and falls back to the POSIX TZ rule
// for times after the last transition.
func (z *Zone) Lookup(t time.Time) ZoneType {
	unix := t.Unix()

	if len(z.transitions) == 0 {
		if len(z.types) > 0 {
			return z.types[0]
		}
		return ZoneType{Abbrev: "UTC"}
	}

	// Binary search for the transition.
	lo, hi := 0, len(z.transitions)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if z.transitions[mid].When <= unix {
			lo = mid + 1
		} else {
			hi = mid
		}
	}

	if lo == 0 {
		// Before the first transition: use the first non-DST type,
		// or type 0 if none.
		for _, zt := range z.types {
			if !zt.IsDST {
				return zt
			}
		}
		return z.types[0]
	}

	if lo == len(z.transitions) && z.extend != nil {
		// After the last transition: use the POSIX TZ rule.
		abbrev, offset, isDST := z.extend.Lookup(unix)
		return ZoneType{Abbrev: abbrev, Offset: offset, IsDST: isDST}
	}

	return z.types[z.transitions[lo-1].Type]
}

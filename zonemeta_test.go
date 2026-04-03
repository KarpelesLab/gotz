package gotz

import (
	"math"
	"testing"
)

func TestMetaNewYork(t *testing.T) {
	z, err := Load("America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	m := z.Meta()
	if m == nil {
		t.Fatal("Meta() is nil")
	}
	if len(m.Countries) == 0 {
		t.Fatal("no countries")
	}
	if m.Countries[0].Code != "US" {
		t.Errorf("country code = %q, want US", m.Countries[0].Code)
	}
	if m.Countries[0].Name == "" {
		t.Error("country name is empty")
	}

	// New York is approximately 40.7128° N, 74.0060° W
	if math.Abs(m.Lat-40.7142) > 0.5 {
		t.Errorf("Lat = %f, want ~40.71", m.Lat)
	}
	if math.Abs(m.Lon-(-74.0060)) > 0.5 {
		t.Errorf("Lon = %f, want ~-74.01", m.Lon)
	}
}

func TestMetaTokyo(t *testing.T) {
	z, err := Load("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	m := z.Meta()
	if m == nil {
		t.Fatal("Meta() is nil")
	}
	if m.Countries[0].Code != "JP" {
		t.Errorf("country code = %q, want JP", m.Countries[0].Code)
	}
	if m.Lat < 35 || m.Lat > 36 {
		t.Errorf("Lat = %f, want ~35.68", m.Lat)
	}
}

func TestMetaMultipleCountries(t *testing.T) {
	// Asia/Dubai covers AE,OM,RE,SC,TF
	z, err := Load("Asia/Dubai")
	if err != nil {
		t.Fatal(err)
	}
	m := z.Meta()
	if m == nil {
		t.Fatal("Meta() is nil")
	}
	if len(m.Countries) < 2 {
		t.Errorf("countries len = %d, want >= 2", len(m.Countries))
	}
	if m.Countries[0].Code != "AE" {
		t.Errorf("first country = %q, want AE", m.Countries[0].Code)
	}
}

func TestMetaUTC(t *testing.T) {
	z, err := Load("UTC")
	if err != nil {
		t.Fatal(err)
	}
	m := z.Meta()
	// UTC has no zone1970.tab entry, so Meta() should return nil.
	if m != nil {
		t.Errorf("UTC Meta() = %+v, want nil", m)
	}
}

func TestParseISO6709(t *testing.T) {
	tests := []struct {
		s        string
		wantLat  float64
		wantLon  float64
	}{
		{"+4030-07400", 40.5, -74.0},
		{"+3431+06912", 34.5167, 69.2},
		{"+352439+1394744", 35.4108, 139.7956},
	}
	for _, tt := range tests {
		lat, lon := parseISO6709(tt.s)
		if math.Abs(lat-tt.wantLat) > 0.01 {
			t.Errorf("parseISO6709(%q) lat = %f, want %f", tt.s, lat, tt.wantLat)
		}
		if math.Abs(lon-tt.wantLon) > 0.01 {
			t.Errorf("parseISO6709(%q) lon = %f, want %f", tt.s, lon, tt.wantLon)
		}
	}
}

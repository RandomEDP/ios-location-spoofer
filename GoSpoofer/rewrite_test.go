package main

import (
	"testing"

	pb "golocationspoofer/pb"
	"google.golang.org/protobuf/proto"
)

// A typical phone request: Wi-Fi BSSIDs with no locations, plus request-only fields.
func TestRewriteBuildsCompleteAnswer(t *testing.T) {
	n := int32(1)
	m := int32(-1)
	req := &pb.AppleWLoc{
		WifiDevices:    []*pb.WifiDevice{{Bssid: "aa:bb:cc:dd:ee:01"}, {Bssid: "aa:bb:cc:dd:ee:02"}},
		NumCellResults: &n,
		NumWifiResults: &m,
		DeviceType:     &pb.DeviceType{OperatingSystem: "iPhone OS26.2/23C55", Model: "iPhone17,1"},
	}
	payload, err := proto.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	lat, lon := IntFromCoord(40.770717), IntFromCoord(-73.957233)
	out, modified := rewriteAppleWLocCoords(payload, lat, lon)
	if modified == 0 {
		t.Fatal("nothing rewritten")
	}
	got := &pb.AppleWLoc{}
	if err := proto.Unmarshal(out, got); err != nil {
		t.Fatalf("rewritten answer doesn't parse: %v", err)
	}
	if got.NumCellResults != nil || got.NumWifiResults != nil || got.DeviceType != nil {
		t.Fatalf("request-only fields left in the answer: %v", got)
	}
	if len(got.WifiDevices) != 2 {
		t.Fatalf("want 2 wifi devices, got %d", len(got.WifiDevices))
	}
	for _, d := range got.WifiDevices {
		l := d.GetLocation()
		if l.GetLatitude() != lat || l.GetLongitude() != lon {
			t.Fatalf("%s: wrong coords %d,%d", d.GetBssid(), l.GetLatitude(), l.GetLongitude())
		}
		if l.GetHorizontalAccuracy() != 39 || l.GetAltitude() != 530 || l.GetVerticalAccuracy() != 1000 {
			t.Fatalf("%s: missing accuracy/altitude fields: %v", d.GetBssid(), l)
		}
	}
}

// If a location is already present (e.g. a real Apple answer), keep its own accuracy and only move it.
func TestRewriteKeepsExistingAccuracy(t *testing.T) {
	a, b, acc := int64(1), int64(2), int64(12)
	req := &pb.AppleWLoc{WifiDevices: []*pb.WifiDevice{{Bssid: "aa:bb:cc:dd:ee:03",
		Location: &pb.Location{Latitude: &a, Longitude: &b, HorizontalAccuracy: &acc}}}}
	payload, _ := proto.Marshal(req)
	lat, lon := IntFromCoord(51.5), IntFromCoord(-0.12)
	out, _ := rewriteAppleWLocCoords(payload, lat, lon)
	got := &pb.AppleWLoc{}
	if err := proto.Unmarshal(out, got); err != nil {
		t.Fatal(err)
	}
	l := got.WifiDevices[0].GetLocation()
	if l.GetLatitude() != lat || l.GetLongitude() != lon || l.GetHorizontalAccuracy() != 12 {
		t.Fatalf("unexpected location %v", l)
	}
}

func TestLocationHosts(t *testing.T) {
	for _, h := range []string{"gs-loc.apple.com:443", "GS-LOC-CN.apple.com", "gsp-ssl.ls.apple.com:443"} {
		if !isLocationHost(h) {
			t.Fatalf("%s should be a location host", h)
		}
	}
	if isLocationHost("www.apple.com:443") {
		t.Fatal("www.apple.com is not a location host")
	}
}

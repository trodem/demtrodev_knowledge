package systeminfo

import "testing"

func TestParseWiFiNetworks(t *testing.T) {
	in := `
Interface name : Wi-Fi
There are 2 networks currently visible.

SSID 1 : HomeNet
    Network type            : Infrastructure
    Authentication          : WPA2-Personal
    Signal                  : 79%

SSID 2 : Guest
    Authentication          : Open
    Signal                  : 42%
`
	got := parseWiFiNetworks(in)
	if len(got) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(got))
	}
	if got[0].SSID != "Guest" || got[1].SSID != "HomeNet" {
		t.Fatalf("unexpected SSIDs: %+v", got)
	}
	if got[1].Signal != "79%" || got[1].Authentication != "WPA2-Personal" {
		t.Fatalf("unexpected network fields: %+v", got[1])
	}
}

func TestParseConnectedSSID(t *testing.T) {
	in := `
Name                   : Wi-Fi
State                  : connected
SSID                   : OfficeNet
BSSID                  : aa:bb:cc:dd:ee:ff
`
	got := parseConnectedSSID(in)
	if got != "OfficeNet" {
		t.Fatalf("expected OfficeNet, got %q", got)
	}
}

func TestParseARPTable(t *testing.T) {
	in := `
Interface: 192.168.1.100 --- 0xb
  Internet Address      Physical Address      Type
  192.168.1.1           aa-bb-cc-dd-ee-ff     dynamic
  192.168.1.10          11-22-33-44-55-66     dynamic
`
	got := parseARPTable(in)
	if len(got) != 2 {
		t.Fatalf("expected 2 neighbors, got %d", len(got))
	}
	if got[0].IP != "192.168.1.1" || got[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("unexpected first row: %+v", got[0])
	}
}

package systeminfo

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Snapshot struct {
	GeneratedAt   time.Time
	System        System
	Memory        Memory
	Disks         []Disk
	Interfaces    []Interface
	ConnectedWiFi string
	WiFiNetworks  []WiFiNetwork
	LANNeighbors  []LANNeighbor
	Warnings      []string
}

type System struct {
	Hostname string
	OS       string
	Arch     string
	CPUCount int
	BootTime time.Time
}

type Memory struct {
	TotalBytes uint64
	FreeBytes  uint64
}

type Disk struct {
	Name      string
	SizeBytes uint64
	FreeBytes uint64
}

type Interface struct {
	Name      string
	Up        bool
	Hardware  string
	Addresses []string
}

type WiFiNetwork struct {
	SSID           string
	Signal         string
	Authentication string
}

type LANNeighbor struct {
	IP   string
	MAC  string
	Type string
}

func Collect() Snapshot {
	s := Snapshot{
		GeneratedAt: time.Now(),
		System: System{
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			CPUCount: runtime.NumCPU(),
		},
	}

	if host, err := os.Hostname(); err == nil {
		s.System.Hostname = host
	} else {
		s.Warnings = append(s.Warnings, fmt.Sprintf("hostname: %v", err))
	}

	ifaces, err := collectInterfaces()
	if err != nil {
		s.Warnings = append(s.Warnings, fmt.Sprintf("interfaces: %v", err))
	} else {
		s.Interfaces = ifaces
	}

	neighbors, err := collectLANNeighbors()
	if err != nil {
		s.Warnings = append(s.Warnings, fmt.Sprintf("arp: %v", err))
	} else {
		s.LANNeighbors = neighbors
	}

	if runtime.GOOS == "windows" {
		if err := collectWindowsSystem(&s); err != nil {
			s.Warnings = append(s.Warnings, err.Error())
		}
	}

	return s
}

func collectInterfaces() ([]Interface, error) {
	netIfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	out := make([]Interface, 0, len(netIfs))
	for _, inf := range netIfs {
		addrs, err := inf.Addrs()
		if err != nil {
			continue
		}
		entry := Interface{
			Name:      inf.Name,
			Up:        inf.Flags&net.FlagUp != 0,
			Hardware:  inf.HardwareAddr.String(),
			Addresses: make([]string, 0, len(addrs)),
		}
		for _, addr := range addrs {
			entry.Addresses = append(entry.Addresses, addr.String())
		}
		if len(entry.Addresses) == 0 {
			continue
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func collectLANNeighbors() ([]LANNeighbor, error) {
	raw, err := runCmd(4*time.Second, "arp", "-a")
	if err != nil {
		return nil, err
	}
	return parseARPTable(raw), nil
}

func collectWindowsSystem(s *Snapshot) error {
	mem, boot, err := windowsMemoryAndBoot()
	if err != nil {
		return fmt.Errorf("windows memory/boot: %w", err)
	}
	s.Memory = mem
	if !boot.IsZero() {
		s.System.BootTime = boot
	}

	disks, err := windowsDisks()
	if err != nil {
		s.Warnings = append(s.Warnings, fmt.Sprintf("windows disks: %v", err))
	} else {
		s.Disks = disks
	}

	connected, err := windowsConnectedWiFi()
	if err != nil {
		s.Warnings = append(s.Warnings, fmt.Sprintf("windows wifi interface: %v", err))
	} else {
		s.ConnectedWiFi = connected
	}

	wifiNetworks, err := windowsWiFiNetworks()
	if err != nil {
		s.Warnings = append(s.Warnings, fmt.Sprintf("windows wifi scan: %v", err))
	} else {
		s.WiFiNetworks = wifiNetworks
	}

	return nil
}

func windowsMemoryAndBoot() (Memory, time.Time, error) {
	const script = "$os=Get-CimInstance Win32_OperatingSystem; [pscustomobject]@{TotalKB=[uint64]$os.TotalVisibleMemorySize; FreeKB=[uint64]$os.FreePhysicalMemory; LastBoot=([datetime]$os.LastBootUpTime).ToUniversalTime().ToString('o')} | ConvertTo-Json -Compress"
	var res struct {
		TotalKB  uint64 `json:"TotalKB"`
		FreeKB   uint64 `json:"FreeKB"`
		LastBoot string `json:"LastBoot"`
	}
	if err := runPowerShellJSON(script, &res); err != nil {
		return Memory{}, time.Time{}, err
	}
	boot := time.Time{}
	if strings.TrimSpace(res.LastBoot) != "" {
		if bt, err := time.Parse(time.RFC3339Nano, res.LastBoot); err == nil {
			boot = bt
		}
	}
	return Memory{
		TotalBytes: res.TotalKB * 1024,
		FreeBytes:  res.FreeKB * 1024,
	}, boot, nil
}

func windowsDisks() ([]Disk, error) {
	const script = "Get-CimInstance Win32_LogicalDisk -Filter \"DriveType=3\" | Select-Object DeviceID,Size,FreeSpace | ConvertTo-Json -Compress"
	var one struct {
		DeviceID  string      `json:"DeviceID"`
		Size      interface{} `json:"Size"`
		FreeSpace interface{} `json:"FreeSpace"`
	}
	if err := runPowerShellJSON(script, &one); err == nil && strings.TrimSpace(one.DeviceID) != "" {
		return []Disk{toDisk(one.DeviceID, one.Size, one.FreeSpace)}, nil
	}

	var many []struct {
		DeviceID  string      `json:"DeviceID"`
		Size      interface{} `json:"Size"`
		FreeSpace interface{} `json:"FreeSpace"`
	}
	if err := runPowerShellJSON(script, &many); err != nil {
		return nil, err
	}
	out := make([]Disk, 0, len(many))
	for _, d := range many {
		out = append(out, toDisk(d.DeviceID, d.Size, d.FreeSpace))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func toDisk(name string, sizeVal, freeVal interface{}) Disk {
	size := parseUintAny(sizeVal)
	free := parseUintAny(freeVal)
	return Disk{Name: name, SizeBytes: size, FreeBytes: free}
}

func windowsConnectedWiFi() (string, error) {
	out, err := runCmd(5*time.Second, "netsh", "wlan", "show", "interfaces")
	if err != nil {
		if isWiFiUnavailableError(err) {
			return "", nil
		}
		return "", err
	}
	return parseConnectedSSID(out), nil
}

func windowsWiFiNetworks() ([]WiFiNetwork, error) {
	out, err := runCmd(10*time.Second, "netsh", "wlan", "show", "networks", "mode=bssid")
	if err != nil {
		if isWiFiUnavailableError(err) {
			return nil, nil
		}
		return nil, err
	}
	return parseWiFiNetworks(out), nil
}

func runPowerShellJSON(script string, out any) error {
	text, err := runCmd(8*time.Second, "powershell", "-NoProfile", "-Command", script)
	if err != nil {
		return err
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("empty output")
	}
	return json.Unmarshal([]byte(text), out)
}

func runCmd(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	b, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("timeout")
	}
	if err != nil {
		return "", fmt.Errorf("%v (%s)", err, strings.TrimSpace(string(b)))
	}
	return string(b), nil
}

func isWiFiUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "is turned off") || strings.Contains(msg, "ausgeschaltet") || strings.Contains(msg, "disattiv")
}

func parseUint(v string) uint64 {
	n, _ := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
	return n
}

func parseUintAny(v interface{}) uint64 {
	switch x := v.(type) {
	case float64:
		if x < 0 {
			return 0
		}
		return uint64(x)
	case string:
		return parseUint(x)
	default:
		return 0
	}
}

func parseConnectedSSID(out string) string {
	lines := strings.Split(out, "\n")
	connected := false
	ssid := ""
	for _, line := range lines {
		t := strings.TrimSpace(line)
		l := strings.ToLower(t)
		if strings.HasPrefix(l, "state") || strings.HasPrefix(l, "stato") {
			val := valueAfterColon(t)
			lv := strings.ToLower(val)
			connected = strings.Contains(lv, "connected") || strings.Contains(lv, "conness")
			continue
		}
		if strings.HasPrefix(l, "ssid") && !strings.HasPrefix(l, "bssid") {
			val := valueAfterColon(t)
			if val != "" {
				ssid = val
			}
		}
	}
	if connected {
		return ssid
	}
	return ""
}

func parseWiFiNetworks(out string) []WiFiNetwork {
	lines := strings.Split(out, "\n")
	var outNet []WiFiNetwork
	var curr WiFiNetwork
	flush := func() {
		if strings.TrimSpace(curr.SSID) != "" {
			outNet = append(outNet, curr)
		}
		curr = WiFiNetwork{}
	}
	for _, line := range lines {
		t := strings.TrimSpace(line)
		l := strings.ToLower(t)
		if strings.HasPrefix(l, "ssid ") && !strings.HasPrefix(l, "bssid") {
			flush()
			curr.SSID = valueAfterColon(t)
			continue
		}
		if strings.HasPrefix(l, "authentication") || strings.HasPrefix(l, "autenticazione") {
			curr.Authentication = valueAfterColon(t)
			continue
		}
		if strings.HasPrefix(l, "signal") || strings.HasPrefix(l, "segnale") {
			curr.Signal = valueAfterColon(t)
			continue
		}
	}
	flush()
	sort.Slice(outNet, func(i, j int) bool {
		return strings.ToLower(outNet[i].SSID) < strings.ToLower(outNet[j].SSID)
	})
	return outNet
}

func valueAfterColon(line string) string {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return ""
	}
	return strings.TrimSpace(line[idx+1:])
}

var arpLineRe = regexp.MustCompile(`^\s*([0-9]{1,3}(?:\.[0-9]{1,3}){3})\s+([0-9a-fA-F:-]{11,})\s+([a-zA-Z]+)\s*$`)

func parseARPTable(out string) []LANNeighbor {
	lines := strings.Split(out, "\n")
	seen := map[string]struct{}{}
	list := []LANNeighbor{}
	for _, line := range lines {
		m := arpLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) != 4 {
			continue
		}
		entry := LANNeighbor{
			IP:   m[1],
			MAC:  strings.ToLower(strings.ReplaceAll(m[2], "-", ":")),
			Type: strings.ToLower(m[3]),
		}
		key := entry.IP + "|" + entry.MAC
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		list = append(list, entry)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].IP < list[j].IP })
	return list
}

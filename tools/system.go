package tools

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	"cli/internal/systeminfo"
)

func RunSystem(_ *bufio.Reader) int {
	s := systeminfo.Collect()

	fmt.Printf("Snapshot: %s\n", s.GeneratedAt.Format(time.RFC3339))
	fmt.Println("== System ==")
	fmt.Printf("Host: %s\n", valueOrDash(s.System.Hostname))
	fmt.Printf("OS: %s/%s\n", s.System.OS, s.System.Arch)
	fmt.Printf("CPU: %d\n", s.System.CPUCount)
	if !s.System.BootTime.IsZero() {
		uptime := time.Since(s.System.BootTime).Round(time.Minute)
		fmt.Printf("Boot time: %s\n", s.System.BootTime.Format(time.RFC3339))
		fmt.Printf("Uptime: %s\n", uptime)
	}
	if s.Memory.TotalBytes > 0 {
		used := s.Memory.TotalBytes - s.Memory.FreeBytes
		fmt.Printf("Memory: %s used / %s total\n", formatBytes(used), formatBytes(s.Memory.TotalBytes))
	}

	fmt.Println("\n== Disks ==")
	if len(s.Disks) == 0 {
		fmt.Println("- none")
	} else {
		for _, d := range s.Disks {
			used := d.SizeBytes - d.FreeBytes
			usedPct := 0.0
			if d.SizeBytes > 0 {
				usedPct = (float64(used) / float64(d.SizeBytes)) * 100
			}
			fmt.Printf("- %s: %s used / %s total (%.1f%%)\n", d.Name, formatBytes(used), formatBytes(d.SizeBytes), usedPct)
		}
	}

	fmt.Println("\n== Interfaces ==")
	if len(s.Interfaces) == 0 {
		fmt.Println("- none")
	} else {
		for _, inf := range s.Interfaces {
			state := "down"
			if inf.Up {
				state = "up"
			}
			addrs := "-"
			if len(inf.Addresses) > 0 {
				addrs = strings.Join(inf.Addresses, ", ")
			}
			fmt.Printf("- %s (%s) mac=%s\n  %s\n", inf.Name, state, valueOrDash(inf.Hardware), addrs)
		}
	}

	fmt.Println("\n== Wi-Fi ==")
	fmt.Printf("Connected: %s\n", valueOrDash(s.ConnectedWiFi))
	if len(s.WiFiNetworks) == 0 {
		fmt.Println("- no networks detected")
	} else {
		for _, net := range s.WiFiNetworks {
			fmt.Printf("- %s | signal=%s | auth=%s\n", valueOrDash(net.SSID), valueOrDash(net.Signal), valueOrDash(net.Authentication))
		}
	}

	fmt.Println("\n== LAN Neighbors (ARP) ==")
	if len(s.LANNeighbors) == 0 {
		fmt.Println("- none")
	} else {
		limit := len(s.LANNeighbors)
		if limit > 25 {
			limit = 25
		}
		for i := 0; i < limit; i++ {
			n := s.LANNeighbors[i]
			fmt.Printf("- %s | %s | %s\n", n.IP, n.MAC, n.Type)
		}
		if len(s.LANNeighbors) > limit {
			fmt.Printf("- ... and %d more\n", len(s.LANNeighbors)-limit)
		}
	}

	if len(s.Warnings) > 0 {
		fmt.Println("\n== Notes ==")
		for _, w := range s.Warnings {
			fmt.Printf("- %s\n", w)
		}
	}
	return 0
}

func formatBytes(n uint64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
		tb = 1024 * gb
	)
	switch {
	case n >= tb:
		return fmt.Sprintf("%.2fTB", float64(n)/float64(tb))
	case n >= gb:
		return fmt.Sprintf("%.2fGB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.2fMB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.2fKB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

func valueOrDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}

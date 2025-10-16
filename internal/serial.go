package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

func ListSerialPorts() ([]string, error) {
	switch runtime.GOOS {
	case "linux":
		return globSerialPorts([]string{
			"/dev/ttyS*",
			"/dev/ttyUSB*",
			"/dev/ttyACM*",
			"/dev/ttyAMA*",
			"/dev/ttyAP*",
			"/dev/tty.*",
			"/dev/cu.*",
		})
	case "darwin":
		return globSerialPorts([]string{
			"/dev/tty.*",
			"/dev/cu.*",
		})
	case "windows":
		ports := make([]string, 0, 32)
		for i := 1; i <= 32; i++ {
			ports = append(ports, fmt.Sprintf("COM%d", i))
		}
		return ports, nil
	default:
		return []string{}, nil
	}
}

func globSerialPorts(patterns []string) ([]string, error) {
	seen := map[string]struct{}{}
	var ports []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if _, ok := seen[match]; ok {
				continue
			}
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			mode := info.Mode()
			if mode&os.ModeDevice == 0 && mode&os.ModeCharDevice == 0 {
				continue
			}
			seen[match] = struct{}{}
			ports = append(ports, match)
		}
	}
	sort.Strings(ports)
	return ports, nil
}

package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	gnet "github.com/shirou/gopsutil/v3/net"
)

// return %ProgramData%/appName or %appdata%/appName
func GetAppDataPath(appName string) string {
	appData := os.Getenv("ProgramData")
	if appData == "" {
		appData, _ = os.UserConfigDir()
	}

	return filepath.Join(appData, "chainnode", appName)
}

func GetStoragePath() string {
	var basePath = os.Getenv("CHAIN_NODE_STORAGE_PATH")
	if basePath == "" {
		basePath = filepath.Join(GetAppDataPath(""), "storage")
	}

	return basePath
}

func GetStorageDrive() string {
	basePath := GetStoragePath()
	return basePath[:len("c:/")]
}

func GetLogPath(appName string, moduleNames ...string) string {
	var basePath = os.Getenv("CHAIN_NODE_LOG_PATH")
	if basePath == "" {
		basePath = GetAppDataPath(appName)
	}

	if len(moduleNames) == 0 {
		basePath = filepath.Join(basePath, "log")
	} else {
		for _, m := range moduleNames {
			basePath = filepath.Join(basePath, m)
		}
		basePath = filepath.Join(basePath, "log")
	}

	logPath := basePath
	if logPath != "" {
		//TryCreateDir(logPath)
	}

	return logPath
}

func TryCreateDir(path string) string {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		log.Fatalf("create directory %s, error: %v", path, err)
	}
	return path
}

func NowString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// GetAllIPAddresses returns a slice of strings in the format "adapter name: ip address",
// excluding loopback interfaces and CIDR notation.
func GetAllIPAddresses() ([]string, error) {
	interfaces, err := gnet.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get network interfaces: %w", err)
	}

	var result []string
	for _, iface := range interfaces {

		for _, addr := range iface.Addrs {
			// Parse IP from CIDR (e.g., "192.168.1.5/24" -> "192.168.1.5")
			ip, _, err := net.ParseCIDR(addr.Addr)
			if err != nil {
				// If not CIDR, try as plain IP
				ip = net.ParseIP(addr.Addr)
				if ip == nil {
					continue // invalid IP, skip
				}
			}

			// Extra safety: skip loopback IPs (in case flag missed)
			if ip.IsLoopback() {
				continue
			}

			// Skip unspecified (0.0.0.0, ::) and multicast addresses
			if ip.IsUnspecified() || ip.IsMulticast() {
				continue
			}

			result = append(result, fmt.Sprintf("%s | %s", iface.Name, ip.String()))
		}
	}

	return result, nil
}

func FindIPByPrefix(addresses []string, prefix string) (string, error) {
	for _, entry := range addresses {
		// Split by ":" to separate adapter name and IP
		// Note: some adapter names may contain colons (e.g., on Windows),
		// but gopsutil typically uses simple names like "eth0", "en0", etc.
		// We split at the last colon + space to be safe, but for simplicity,
		// we assume format is "name: ip" with single colon+space.
		parts := strings.SplitN(entry, " | ", 2)
		if len(parts) != 2 {
			continue // skip malformed entries
		}
		ipStr := parts[1]

		// Check if IP starts with the given prefix
		if strings.HasPrefix(ipStr, prefix) {
			return ipStr, nil
		}
	}
	return "", fmt.Errorf("no IP address found with prefix %q", prefix)
}

func AnyFromFile(dataFile string, data interface{}) error {
	jsonBlob, err := os.ReadFile(dataFile)
	if err != nil {
		return fmt.Errorf("couldn't read file: %w", err)
	}

	err = json.Unmarshal(jsonBlob, data)
	if err != nil {
		return fmt.Errorf("invalid json: %s", err)
	}

	return nil
}

func DeviceID() string {
	return UUID()
}

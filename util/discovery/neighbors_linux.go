package discovery

import "os"

func readNeighbors() (map[string]string, error) {
	b, err := os.ReadFile("/proc/net/arp")
	return parseNeighbors(string(b)), err
}

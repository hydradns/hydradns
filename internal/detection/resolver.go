package detection

import (
	"bufio"
	"errors"
	"net"
	"os"
	"strings"
)

// SystemResolver returns the first nameserver entry found in a resolv.conf file.
func SystemResolver(path string) (net.IP, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" {
			ip := net.ParseIP(fields[1])
			if ip != nil {
				return ip, nil
			}
		}
	}

	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return nil, errors.New("System resolver not found at " + path)
}

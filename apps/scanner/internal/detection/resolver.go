package detection

import (
	"bufio"
	"errors"
	"net"
	"os"
	"strings"

	"github.com/PhantomDNS/scanner/internal/checks"
)

// SystemResolver returns the first nameserver entry found in a resolv.conf file.
func SystemResolver(path string) (checks.Resolver, error) {
	f, err := os.Open(path)
	if err != nil {
		return checks.Resolver{}, err
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
				return checks.Resolver{IP: ip, Port: 53}, nil
			}
		}
	}

	if scanner.Err() != nil {
		return checks.Resolver{}, scanner.Err()
	}

	return checks.Resolver{}, errors.New("System resolver not found at " + path)
}

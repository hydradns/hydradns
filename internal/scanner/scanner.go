package scanner

import (
	"context"
	"fmt"

	"github.com/PhantomDNS/scanner/internal/checks"
	"github.com/PhantomDNS/scanner/internal/detection"
)

type ScannerResult struct {
	Resolver string
	Checks   []checks.Result
}

// 1. Call resolver detection
// 2. Run UDP check
// 3. Collect results
// 4. Return a struct
type Scanner struct {

	// maybe implement checks later
	// Checks   []checks.Check
}

func (s *Scanner) Scan(ctx context.Context) (ScannerResult, error) {
	fmt.Print("Scanning ...")
	// run resolver detection
	resolver, err := detection.SystemResolver("/etc/resolv.conf")
	if err != nil {
		return ScannerResult{}, err
	}

	udpCheck := checks.UDPResolutionCheck{}

	result, err := udpCheck.Run(ctx, resolver)

	if err != nil {
		return ScannerResult{}, err
	}

	return ScannerResult{
		Resolver: resolver.String(),
		Checks:   []checks.Result{result},
	}, nil
}

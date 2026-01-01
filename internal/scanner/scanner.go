package scanner

import (
	"context"
	"fmt"

	"github.com/PhantomDNS/scanner/internal/checks"
	"github.com/PhantomDNS/scanner/internal/detection"
)

type ScannerResult struct {
	Resolver string
	Checks   map[string]checks.Result
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

	checkSet := []checks.Check{
		&checks.UDPResolutionCheck{},
	}

	results := make(map[string]checks.Result)

	for _, check := range checkSet {
		res, err := check.Run(ctx, resolver)
		if err != nil {
			results[check.Name()] = checks.Result{
				Status: checks.StatusError,
				Reason: err.Error(),
			}
			continue
		}
		results[check.Name()] = res
	}
	return ScannerResult{
		Resolver: fmt.Sprintf("%s:%d", resolver.IP.String(), resolver.Port),
		Checks:   results,
	}, nil
}

package checks

import (
	"context"
	"net"
	"time"

	"github.com/miekg/dns"
)

type UDPResolutionCheck struct {
	Domain string
}

type DNSResult struct {
	Domain   string
	Resolver net.IP
	QType    string

	Success bool
	Error   error
	RTT     time.Duration

	Answers []string
}

func (c *UDPResolutionCheck) Run(ctx context.Context, resolver net.IP) (Result, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	domain := c.Domain
	if domain == "" {
		domain = "www.google.com"
	}
	result := &DNSResult{
		Domain:   domain,
		Resolver: resolver,
		QType:    "A",
	}

	// build dns message
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	client := &dns.Client{
		Net: "udp",
	}

	start := time.Now()

	// Respect context cancellation
	type response struct {
		r   *dns.Msg
		err error
	}

	ch := make(chan response, 1)

	go func() {
		r, _, err := client.Exchange(msg, net.JoinHostPort(resolver.String(), "53"))
		select {
		case ch <- response{r: r, err: err}:
		case <-ctx.Done():
		}
	}()

	select {
	case <-ctx.Done():
		return Result{
			Status:   "fail",
			Evidence: map[string]interface{}{"error": ctx.Err().Error()},
		}, nil
	case resp := <-ch:
		result.Success = resp.err == nil
		if resp.err != nil {
			result.Error = resp.err
		}
		if resp.r != nil {
			for _, ans := range resp.r.Answer {
				if a, ok := ans.(*dns.A); ok {
					result.Answers = append(result.Answers, a.A.String())
				}
			}
		}
		result.RTT = time.Since(start)
	}

	status := "fail"
	if result.Success {
		status = "pass"
	}

	return Result{
		Status: status,
		Evidence: map[string]interface{}{
			"domain":   domain,
			"resolver": resolver.String(),
			"qtype":    "A",
			"rtt_ms":   result.RTT.Milliseconds(),
			"answers":  result.Answers,
		},
	}, nil
}

func (c *UDPResolutionCheck) Name() string {
	return "udp_resolution"
}

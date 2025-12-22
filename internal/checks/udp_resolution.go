package checks

import (
	"context"
	"net"
	"time"

	"github.com/miekg/dns"
)

type DNSResolutionResult struct {
	Domain   string
	Resolver net.IP
	QType    string

	Success bool
	Error   error
	RTT     time.Duration

	Answers []string
}

func UDPResolutionCheck(ctx context.Context, resolver net.IP) DNSResolutionResult {
	domain := "www.google.com"
	result := &DNSResolutionResult{
		Domain:   domain,
		Resolver: resolver,
		QType:    "A",
	}

	// build dns message
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	msg.RecursionDesired = true

	client := &dns.Client{
		Net:     "udp",
		Timeout: 2 * time.Second,
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
		ch <- response{r: r, err: err}
	}()

	select {
	case <-ctx.Done():
		return DNSResolutionResult{}
	case resp := <-ch:
		result.Success = resp.err == nil
		result.Error = resp.err
		if resp.r != nil {
			for _, ans := range resp.r.Answer {
				result.Answers = append(result.Answers, ans.String())
			}
		}
		result.RTT = time.Since(start)
	}
	return *result
}

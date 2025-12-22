package main

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PhantomDNS/scanner/internal/checks"
	"github.com/PhantomDNS/scanner/internal/scanner"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	resolver, err := SystemResolver("/etc/resolv.conf")

	r.GET("/scan", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		select {
		case <-ctx.Done():
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "Request timed out"})
			return
		default:
		}

		udpResolution := checks.UDPResolutionCheck(ctx, resolver)

		resp := gin.H{
			"message":           "Scan endpoint",
			"detected_resolver": resolver.String(),
			"domain":            udpResolution.Domain,
			"qtype":             udpResolution.QType,
			"success":           udpResolution.Success,
			"error":             "",
			"rtt":               udpResolution.RTT.String(),
			"answers":           udpResolution.Answers,
		}
		if udpResolution.Error != nil {
			resp["error"] = udpResolution.Error.Error()
		}
		c.JSON(http.StatusOK, resp)
	})

	if err != nil {
		panic(err)
	}

	println("System resolver:", resolver.String())

	if err != nil {
		panic(err)
	}

	r.Run(":8080")

	s := &scanner.Scanner{}
	_, _ = s.Scan(context.Background())
}

func SystemResolver(path string) (net.IP, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" {
			ip := net.ParseIP(fields[1])
			if ip != nil {
				return ip, nil
			}
		}
	}

	return nil, nil
}

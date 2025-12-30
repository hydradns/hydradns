package main

import (
	"context"
	"net/http"
	"time"

	"github.com/PhantomDNS/scanner/internal/scanner"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/scan", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		s := &scanner.Scanner{}
		result, err := s.Scan(ctx)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		udpResolution := result.Checks[0]

		resp := gin.H{
			"resolver": result.Resolver,
			"checks":   udpResolution,
		}

		c.JSON(http.StatusOK, resp)
	})

	r.Run(":8080")
}

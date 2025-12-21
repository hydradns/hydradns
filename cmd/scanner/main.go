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

		select {
		case <-ctx.Done():
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "Request timed out"})
			return
		default:
		}

		c.JSON(http.StatusOK, gin.H{"message": "Scan endpoint"})
	})

	r.Run(":8080")

	s := &scanner.Scanner{}
	_, _ = s.Scan(context.Background())
}

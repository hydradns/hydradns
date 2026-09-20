package main

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hydradns/hydradns/apps/scanner/internal/scanner"
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

		resp := gin.H{
			"resolver": result.Resolver,
			"checks":   result.Checks,
		}

		c.JSON(http.StatusOK, resp)
	})

	r.Run(":8080")
}

package main

import (
	"context"
	"net/http"

	"github.com/PhantomDNS/scanner/internal/scanner"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/scan", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Scan endpoint"})
	})

	r.Run(":8080")

	s := &scanner.Scanner{}
	_, _ = s.Scan(context.Background())
}

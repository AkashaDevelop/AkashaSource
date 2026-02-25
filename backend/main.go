package main

import (
	"flag"
	"fmt"
	"log"

	"STfreApi/common"
	"STfreApi/model"
	"STfreApi/router"

	"github.com/gin-gonic/gin"
)

func main() {
	// Parse command line arguments
	var port int
	var driver string
	var dsn string

	flag.IntVar(&port, "port", 8080, "Server port")
	flag.StringVar(&driver, "driver", "sqlite", "Database driver (sqlite, mysql, postgres)")
	flag.StringVar(&dsn, "dsn", "akasha.db", "Database DSN")
	flag.Parse()

	// Initialize Database
	log.Printf("Initializing database with driver: %s", driver)
	common.InitDB(driver, dsn)

	// Load Options
	model.InitOptions()

	// Initialize Gin
	r := gin.Default()

	// Set API Router
	router.SetApiRouter(r)

	// Basic Ping Route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Akasha is running",
			"driver":  driver,
		})
	})

	// Start Server
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

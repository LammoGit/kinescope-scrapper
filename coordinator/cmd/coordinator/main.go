package main

import (
	"context"
	"coordinator/internal/api"
	"coordinator/internal/app"
	st "coordinator/internal/storage"
	"coordinator/pkg/sse"
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
)

var (
	port = flag.Uint("port", 8080, "Port of the coordinator")
)

// validateFlags checks values of the flags and returns error on invalid values
func validateFlags() error {
	if 0 > *port || *port > 65535 {
		return fmt.Errorf("port isn't in the valid range (0-65535): %d", *port)
	}
	return nil
}

func main() {
	// Parse and validate the flags
	flag.Parse()
	if err := validateFlags(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	ctx := context.Background()

	hub := sse.NewHub()
	storage, err := st.NewStorage()
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	app, err := app.New(ctx, hub, storage)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	// Create a new router and assign it the Recovery middleware
	router := gin.New()
	router.Use(gin.Recovery())

	// Load API routes
	api.LoadRouter(router.Group("/api"), app)

	// Run server on the given port
	router.Run(fmt.Sprintf(":%d", *port))
}

package main

import (
	"coordinator/internal/api"
	"coordinator/internal/api/stream"
	jdist "coordinator/internal/job-distributor"
	st "coordinator/internal/storage"
	"flag"
	"fmt"

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

// Global variables
var jd = jdist.New()
var servers = &st.Servers{
	Servers: make(map[string]*st.Server),
}
var hub = stream.NewHub()

func main() {
	// Parse and validate the flags
	flag.Parse()
	if err := validateFlags(); err != nil {
		fmt.Println(err)
		return
	}

	// Create a new router and assign it the Recovery middleware
	router := gin.New()
	router.Use(gin.Recovery())

	// Load API routes
	api.LoadRouter(router.Group("/api"), hub, jd, servers)

	// Run server on the given port
	router.Run(fmt.Sprintf(":%d", *port))
}

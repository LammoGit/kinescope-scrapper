package main

import (
	"coordinator/internal/api"
	"coordinator/internal/api/stream"
	jdist "coordinator/internal/job-distributor"
	st "coordinator/internal/storage"
	"coordinator/internal/ui"
	"flag"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"
)

var (
	port       = flag.Uint("port", 8080, "Port of the coordinator")
	configPath = flag.String("config", "./config.yaml", "Path to the YAML config file")
)

// Config type used for parsing and storing config
type Config struct {
	Job struct {
		Range struct {
			From uint `yaml:"from"`
			To   uint `yaml:"to"`
		} `yaml:"range"`
		RequestTimeoutSeconds    uint `yaml:"request_timeout_seconds"`
		MaxRetries               uint `yaml:"max_retries"`
		GlobalRateLimitRPS       uint `yaml:"global_rate_limit_rps"`
		StopOnConsecutiveMissing uint `yaml:"stop_on_consecutive_missing"`
	} `yaml:"job"`
	Servers []st.ServerInfo `yaml:"servers"`
}

// validateFlags checks values of the flags and returns error on invalid values
func validateFlags() error {
	if 0 > *port || *port > 65535 {
		return fmt.Errorf("port isn't in the valid range (0-65535): %d", *port)
	}
	return nil
}

// Global variables for storing and intercating with jobs and workers
var jd = jdist.New()
var servers = &st.Servers{
	Servers: make(map[string]*st.Server),
}

func main() {
	// Parse and validate the flags
	flag.Parse()
	if err := validateFlags(); err != nil {
		fmt.Println(err)
		return
	}

	// Read bytes of the config file
	configData, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Parse data from the config file
	var config Config
	err = yaml.Unmarshal(configData, &config)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Create a new router and assign it the Recovery middleware
	router := gin.New()
	router.Use(gin.Recovery())

	// Create SSE hub
	hub := stream.NewHub()

	// Load static files
	router.LoadHTMLGlob("templates/*")
	router.Static("/static", "./static")

	// Load routes
	api.LoadRouter(router.Group("/api"), hub, jd, servers)
	ui.LoadRouter(router.Group("/"))

	// Run server on the given port
	router.Run(fmt.Sprintf(":%d", *port))
}

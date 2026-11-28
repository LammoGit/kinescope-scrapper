package main

import (
	"context"
	"coordinator/internal/api"
	"coordinator/internal/app"
	st "coordinator/internal/storage"
	"coordinator/pkg/sse"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	port = flag.Uint("port", 8080, "Port of the coordinator")
)

// validateFlags checks values of the flags and returns error on invalid values
func validateFlags() error {
	if *port > 65535 {
		return fmt.Errorf("port isn't in the valid range (0-65535): %d", *port)
	}
	return nil
}

// setupLogging initializes the default structured logger
func setupLogging() {
	// Set handler log level
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	opts := &slog.HandlerOptions{Level: level}

	// Create handler with "json" or "text" format
	var handler slog.Handler
	if strings.EqualFold(os.Getenv("LOG_FORMAT"), "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	// Set handler as default
	slog.SetDefault(slog.New(handler))
}

func main() {
	setupLogging()

	// Parse and validate the flags
	flag.Parse()
	if err := validateFlags(); err != nil {
		slog.Error("Invalid flags", "error", err)
		os.Exit(1)
	}

	// Setup context to gracefully shutdown on SIGINT and SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize SSE hub and storage variables
	hub := sse.NewHub()
	storage, err := st.NewStorage()
	if err != nil {
		slog.Error("Failed to initialize storage", "error", err)
		os.Exit(2)
	}
	slog.Info("Storage initialized", "driver", fmt.Sprintf("%T", storage))

	// Initialize application logic variable
	app, err := app.New(ctx, hub, storage)
	if err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(2)
	}
	slog.Info("Application initialized")

	// Create a new router and assign it the Recovery middleware and custom logger
	router := gin.New()
	router.Use(gin.Recovery(), api.RequestLogger())

	// Load API routes
	api.LoadRouter(router.Group("/api"), app)
	slog.Info("API routes loaded")

	// Connect HTTP server to context and gin router
	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{
		Addr:    addr,
		Handler: http.NewCrossOriginProtection().Handler(router),
		// Make ctx parent context of all request contexts
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}

	// Run server on the given port
	go func() {
		slog.Info("Coordinator is listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed", "error", err)
			stop()
		}
	}()

	// Waiting for application to shutdown gracefully
	<-ctx.Done()
	slog.Info("Shutting down coordinator gracefully...")

	// Shutdown context with timeout of 15 seconds
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Gracefully shutdown HTTP server
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server forced shutdown", "error", err)
	}

	// Gracefully shutdown storage
	if err := storage.Shutdown(shutdownCtx); err != nil {
		slog.Error("Storage forced shutdown", "error", err)
	}

	// Gracefully shutdown application
	if err := app.Shutdown(shutdownCtx); err != nil {
		slog.Error("Application forced shutdown", "error", err)
	}

	slog.Info("Coordinator stopped cleanly")
}

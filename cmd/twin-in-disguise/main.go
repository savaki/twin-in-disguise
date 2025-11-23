// Copyright 2025 Matt Ho
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/savaki/twin-in-disguise/server"
	"github.com/urfave/cli/v2"
	"google.golang.org/api/option"
)

var version = "dev"

// loggingMiddleware wraps an http.Handler to log all requests with status codes
func loggingMiddleware(next http.Handler, debug bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a response writer wrapper to capture the status code
		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Call the next handler
		next.ServeHTTP(wrapper, r)

		// Log requests only in debug mode
		if debug {
			if wrapper.statusCode == http.StatusNotFound {
				log.Printf("404 Not Found: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
			} else {
				log.Printf("%d: %s %s from %s", wrapper.statusCode, r.Method, r.URL.Path, r.RemoteAddr)
			}
		} else if wrapper.statusCode == http.StatusNotFound {
			// Always log 404s even when not in debug mode
			log.Printf("404 Not Found: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		}
	})
}

// responseWriterWrapper wraps http.ResponseWriter to capture the status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func main() {
	app := &cli.App{
		Name:    "twin-in-disguise",
		Usage:   "Anthropic → Gemini proxy server",
		Version: version,
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Usage:   "HTTP server port",
				EnvVars: []string{"PORT"},
				Value:   8080,
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Usage:   "Enable verbose logging",
				EnvVars: []string{"VERBOSE"},
			},
			&cli.BoolFlag{
				Name:    "debug",
				Usage:   "Enable debug logging (shows Gemini API calls)",
				EnvVars: []string{"DEBUG"},
			},
		},
		Action: runServer,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runServer(c *cli.Context) error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("GEMINI_API_KEY environment variable is required")
	}

	port := c.Int("port")
	verbose := c.Bool("verbose")
	debug := c.Bool("debug")

	ctx := context.Background()
	return startProxyServer(ctx, apiKey, port, verbose, debug)
}

func startProxyServer(ctx context.Context, apiKey string, port int, verbose, debug bool) error {
	// Initialize Gemini client
	geminiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return fmt.Errorf("failed to create Gemini client: %w", err)
	}
	defer geminiClient.Close()

	// Create server with API key for thought signature support
	srv := server.NewWithAPIKey(geminiClient, apiKey)
	srv.SetDebug(debug)

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/messages", srv.HandleMessages)

	// Wrap with logging middleware
	handler := loggingMiddleware(mux, debug)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for server to be ready
	healthEndpoint := fmt.Sprintf("http://localhost:%d/v1/messages", port)
	readyTimeout := time.After(5 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case err := <-serverErr:
			return fmt.Errorf("server failed to start: %w", err)
		case <-readyTimeout:
			return fmt.Errorf("server failed to start listening within 5 seconds")
		case <-ticker.C:
			// Try to connect to verify server is listening
			client := &http.Client{Timeout: 50 * time.Millisecond}
			// Use HEAD to avoid errors (POST would fail without body)
			req, _ := http.NewRequest(http.MethodHead, healthEndpoint, nil)
			if resp, err := client.Do(req); err == nil {
				resp.Body.Close()
				// Server is ready - print setup instructions
				printSetupInstructions(port, verbose, debug)
				goto serverRunning
			}
		}
	}

serverRunning:
	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\nShutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	log.Println("Server stopped")
	return nil
}

func printSetupInstructions(port int, verbose, debug bool) {
	fmt.Println()
	fmt.Println("🚀 Anthropic → Gemini Proxy Server")
	fmt.Printf("   Running on http://localhost:%d\n", port)
	fmt.Println()

	if verbose {
		log.Println("Verbose logging enabled")
	}
	if debug {
		log.Println("Debug logging enabled")
	}

	fmt.Println("Enter these in your shell:")
	fmt.Println()
	fmt.Println("  # Note: ANTHROPIC_BASE_URL requires a secure HTTPS connection (e.g. via ngrok)")
	fmt.Println("  # export ANTHROPIC_BASE_URL=https://your-ngrok-url")
	fmt.Println("  export ANTHROPIC_AUTH_TOKEN=test")
	fmt.Println("  export ANTHROPIC_MODEL=\"gemini-3-pro-preview\"")
	fmt.Println("  export ANTHROPIC_DEFAULT_OPUS_MODEL=\"gemini-3-pro-preview\"")
	fmt.Println("  export ANTHROPIC_DEFAULT_SONNET_MODEL=\"gemini-3-pro-preview\"")
	fmt.Println("  export ANTHROPIC_DEFAULT_HAIKU_MODEL=\"gemini-2.0-flash\"")
	fmt.Println("  export CLAUDE_CODE_SUBAGENT_MODEL=\"gemini-3-pro-preview\"")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop the server")
	fmt.Println()
}

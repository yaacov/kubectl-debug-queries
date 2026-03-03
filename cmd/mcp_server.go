package cmd

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	mcpserver "github.com/yaacov/kubectl-debug-queries/mcp"
	"k8s.io/klog/v2"
)

var (
	sseMode  bool
	port     string
	host     string
	certFile string
	keyFile  string
)

var mcpServerCmd = &cobra.Command{
	Use:   "mcp-server",
	Short: "Start the MCP (Model Context Protocol) server",
	Long: `Start the MCP (Model Context Protocol) server for kubectl-debug-queries.

This server provides AI assistants with access to Kubernetes resources, logs, and events.

Modes:
  Default: Stdio mode for AI assistant integration
  --sse:   HTTP server mode with optional TLS

Security:
  --cert-file:   Path to TLS certificate file (enables TLS when both cert and key provided)
  --key-file:    Path to TLS private key file (enables TLS when both cert and key provided)

SSE Mode Authentication (HTTP Headers):
  Authorization: Bearer <token>    - bearer token for Kubernetes auth
  X-Kubernetes-Server: <url>       - Kubernetes API server URL

  Precedence: HTTP headers (per-session) > CLI flags (--token/--server) > kubeconfig

Quick Setup for AI Assistants:

Claude Desktop: claude mcp add kubectl-debug-queries kubectl debug-queries mcp-server
Cursor IDE: Settings → MCP → Add Server (Name: kubectl-debug-queries, Command: kubectl, Args: debug-queries mcp-server)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		if sseMode {
			return runSSE(ctx, sigChan)
		}
		return runStdio(ctx, sigChan)
	},
}

func runSSE(ctx context.Context, sigChan chan os.Signal) error {
	addr := net.JoinHostPort(host, port)

	innerHandler := mcpsdk.NewSSEHandler(func(req *http.Request) *mcpsdk.Server {
		return mcpserver.CreateServer(req.Header)
	}, nil)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if auth := r.Header.Get("Authorization"); auth != "" {
				scheme := "unknown"
				if parts := strings.SplitN(auth, " ", 2); len(parts) > 0 {
					scheme = parts[0]
				}
				klog.V(2).Infof("[auth] POST request with Authorization: %s [REDACTED]", scheme)
			} else {
				klog.V(2).Info("[auth] POST request with NO Authorization header")
			}
		}
		innerHandler.ServeHTTP(w, r)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	errChan := make(chan error, 1)
	go func() {
		useTLS := certFile != "" && keyFile != ""
		if useTLS {
			klog.Infof("Starting kubectl-debug-queries MCP server with TLS on %s", addr)
			errChan <- server.ListenAndServeTLS(certFile, keyFile)
		} else {
			klog.Infof("Starting kubectl-debug-queries MCP server on %s", addr)
			errChan <- server.ListenAndServe()
		}
	}()

	select {
	case err := <-errChan:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		klog.Info("Context cancelled, shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			klog.Errorf("Server shutdown error: %v", err)
		}
	case <-sigChan:
		klog.Info("Shutting down server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			klog.Errorf("Server shutdown error: %v", err)
		}
	}
	return nil
}

func runStdio(ctx context.Context, sigChan chan os.Signal) error {
	server := mcpserver.CreateServer(nil)

	klog.V(1).Info("Starting kubectl-debug-queries MCP server in stdio mode")

	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Run(ctx, &mcpsdk.StdioTransport{})
	}()

	select {
	case err := <-errChan:
		return err
	case <-sigChan:
		klog.V(1).Info("Shutting down server...")
		return nil
	}
}

func init() {
	mcpServerCmd.Flags().BoolVar(&sseMode, "sse", false, "Run in SSE (Server-Sent Events) mode over HTTP")
	mcpServerCmd.Flags().StringVar(&port, "port", "9091", "Port to listen on for SSE mode")
	mcpServerCmd.Flags().StringVar(&host, "host", "0.0.0.0", "Host address to bind to for SSE mode")
	mcpServerCmd.Flags().StringVar(&certFile, "cert-file", "", "Path to TLS certificate file")
	mcpServerCmd.Flags().StringVar(&keyFile, "key-file", "", "Path to TLS private key file")
	rootCmd.AddCommand(mcpServerCmd)
}

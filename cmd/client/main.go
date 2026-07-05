// Package main is the example client entrypoint that calls the server via pkg/api.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/piwriw/oas-go-template/pkg/api"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("client: %v", err)
	}
}

func run() error {
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8000"
	}

	cw, err := api.NewClientWithResponses(serverURL)
	if err != nil {
		return fmt.Errorf("new client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := cw.GetHealthWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("get health: %w", err)
	}

	switch {
	case resp.JSON200 != nil:
		version := "<nil>"
		if resp.JSON200.Version != nil {
			version = *resp.JSON200.Version
		}
		fmt.Printf("health: status=%s version=%s\n", resp.JSON200.Status, version)
		return nil
	case resp.JSON500 != nil:
		fmt.Printf("health: unhealthy code=%d message=%s\n", resp.JSON500.Code, resp.JSON500.Message)
		return nil
	default:
		return fmt.Errorf("unexpected response (HTTP %d, body=%q): %w", resp.StatusCode(), string(resp.Body), errors.New("non-2xx"))
	}
}

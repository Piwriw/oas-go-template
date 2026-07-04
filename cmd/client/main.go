package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/piwriw/oas-go-template/pkg/api"
)

func main() {
	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	cw, err := api.NewClientWithResponses(serverURL)
	if err != nil {
		log.Fatalf("new client: %v", err)
	}
	_ = cw // also see api.NewClient for raw *http.Response usage

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := cw.GetHealthWithResponse(ctx)
	if err != nil {
		log.Fatalf("get health: %v", err)
	}

	switch {
	case resp.JSON200 != nil:
		version := "<nil>"
		if resp.JSON200.Version != nil {
			version = *resp.JSON200.Version
		}
		fmt.Printf("health: status=%s version=%s\n", resp.JSON200.Status, version)
	case resp.JSON500 != nil:
		fmt.Printf("health: unhealthy code=%s message=%s\n", resp.JSON500.Code, resp.JSON500.Message)
	default:
		fmt.Printf("health: unexpected response (HTTP %d, body=%q)\n", resp.StatusCode(), string(resp.Body))
	}
}

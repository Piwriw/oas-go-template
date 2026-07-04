package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/piwriw/oas-go-template/internal/api"
	"github.com/piwriw/oas-go-template/internal/config"
	"github.com/piwriw/oas-go-template/internal/handler"
	"github.com/piwriw/oas-go-template/internal/middleware"
)

func main() {
	cfg, err := config.NewFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	gin.SetMode(cfg.GinMode)

	h := handler.New()
	strictHandler := api.NewStrictHandler(h, nil)

	r := gin.New()
	r.Use(gin.Recovery(), middleware.Logger())
	api.RegisterHandlers(r, strictHandler)

	log.Printf("server listening on %s (mode=%s)", cfg.HTTPAddr, cfg.GinMode)
	if err := http.ListenAndServe(cfg.HTTPAddr, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}

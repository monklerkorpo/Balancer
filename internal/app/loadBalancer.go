package app

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mk/loadBalancer/internal/config"
	"github.com/mk/loadBalancer/internal/server"
)

func Run() {
    cfgPath := flag.String("config", "configs/config.yaml", "path to config file")
    flag.Parse()

    cfg, err := config.Load(*cfgPath)
    if err != nil {
        log.Fatalf("config load error: %v", err)
    }

    srv, err := server.New(cfg)
    if err != nil {
        log.Fatalf("server init error: %v", err)
    }

    // Запускаем сервер в фоне
    go func() {
        if err := srv.Start(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("server run error: %v", err)
        }
    }()

    // Ждём сигналов
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
    <-quit

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("server shutdown error: %v", err)
    }
}

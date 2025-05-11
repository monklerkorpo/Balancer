package server

import (
	"context"
	"net/http"
	"strconv"
	"time"

	_ "modernc.org/sqlite"

	"github.com/gorilla/mux"
	"github.com/mk/loadBalancer/internal/api"
	"github.com/mk/loadBalancer/internal/balancer"
	"github.com/mk/loadBalancer/internal/config"
	"github.com/mk/loadBalancer/internal/proxy"
	"github.com/mk/loadBalancer/internal/ratelimiter"
	"github.com/mk/loadBalancer/internal/storage"
	"go.uber.org/zap"
)

// Server представляет HTTP-сервер приложения
type Server struct {
	httpServer *http.Server
	logger     *zap.SugaredLogger
}

// New создает новый экземпляр Server
func New(appConfig *config.Config) (*Server, error) {
    // Инициализация логгера и других компонентов
    baseLogger, err := zap.NewProduction()
    if err != nil {
        return nil, err
    }
    sugarLogger := baseLogger.Sugar().With("component", "server")

    // Инициализация базы данных и других компонентов
    clientRepository, err := storage.NewSQLiteClientRepo(appConfig.DatabasePath)
    if err != nil {
        sugarLogger.Errorf("Failed to initialize database: %v", err)
        return nil, err
    }

    rateLimiter := ratelimiter.NewRateLimiter(
        appConfig.RateLimit.Capacity,
        appConfig.RateLimit.RefillRate,
        clientRepository,            
        sugarLogger,
    )

    // Пул бекендов
    backendPool := balancer.NewServerPool(appConfig.Backends)

    // Стратегия балансировки
    strategy, err := balancer.StrategyFactory(appConfig.Strategy)
    if err != nil {
        sugarLogger.Errorf("Failed to create balancing strategy: %v", err)
        return nil, err
    }
    backendPool.SetStrategy(strategy)

    // Настройка маршрутов
    router := mux.NewRouter()

    // 1. Регистрируем API маршруты ДО прокси
    apiHandler := api.NewClientHandler(clientRepository, rateLimiter, sugarLogger)
    apiRouter := router.PathPrefix("/clients").Subrouter() // Это должно быть перед прокси маршрутом
    apiHandler.RegisterRoutes(apiRouter)

    // 2. Настройка прокси для всех остальных запросов
    proxyHandler := proxy.NewProxyHandler(backendPool, rateLimiter, sugarLogger)
    wrappedHandler := ratelimiter.RateLimitMiddleware(rateLimiter, sugarLogger)(proxyHandler)
    router.PathPrefix("/").Handler(wrappedHandler)

    // Создаем HTTP сервер
    httpServer := &http.Server{
        Addr:    ":" + strconv.Itoa(appConfig.Port),
        Handler: router,
        ReadTimeout:  5 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  15 * time.Second,
    }

    return &Server{
        httpServer: httpServer,
        logger:     sugarLogger,
    }, nil
}


// Start запускает сервер
func (s *Server) Start() error {
	s.logger.Infof("Server starting on %s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Errorf("Server failed: %v", err)
		return err
	}
	return nil
}

// Shutdown корректно останавливает сервер
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Server shutting down")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Errorf("Server shutdown error: %v", err)
		return err
	}
	s.logger.Info("Server stopped")
	return nil
}

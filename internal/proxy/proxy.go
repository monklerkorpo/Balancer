// Пакет proxy реализует HTTP-прокси-обработчик, который:
// 1. Получает следующий доступный backend из ServerPool (с учетом алгоритма балансировки).
// 2. Проксирует запрос к выбранному backend-серверу.
// 3. Прокидывает IP клиента через X-Real-IP и X-Forwarded-For.
// 4. Обрабатывает ошибки при недоступности backend'ов и уменьшает активные подключения.
//
// 
//

package proxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/mk/loadBalancer/internal/balancer"        // Пакет с реализацией пулов backend'ов и логики балансировки
	"github.com/mk/loadBalancer/internal/ratelimiter"     // Пакет с middleware и логикой ограничения скорости
	"go.uber.org/zap"
)

// ProxyHandler — основной HTTP-обработчик, который проксирует входящие запросы на backend'ы
type ProxyHandler struct {
    BackendPool   *balancer.ServerPool          // Пул backend-серверов с балансировкой нагрузки
    RateLimiter   *ratelimiter.RateLimiter      // Rate Limiter (не используется напрямую, так как подключается как middleware)
    Logger        *zap.SugaredLogger            // Логгер
}

// NewProxyHandler — конструктор ProxyHandler
func NewProxyHandler(pool *balancer.ServerPool, limiter *ratelimiter.RateLimiter, logger *zap.SugaredLogger) *ProxyHandler {
    return &ProxyHandler{
        BackendPool: pool,
        RateLimiter: limiter,
        Logger:      logger,
    }
}

// ServeHTTP — реализация интерфейса http.Handler
// Выполняет:
// - игнор favicon.ico потому что это лишний шум + он не нужен для логики приложения и обычно не должен проксироваться или учитываться в rate limiter'е, логах или подсчете активных соединений.
// - выбирает backend
// - создает reverse proxy
// - прокидывает IP клиента
// - логирует и управляет соединениями
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	
	clientIP := getClientIP(r) // Извлекаем IP клиента для логирования и прокидывания

	backend := h.BackendPool.NextBackend()
	if backend == nil {
		http.Error(w, "no available backends", http.StatusServiceUnavailable)
		return
	}

	targetURL, _ := url.Parse(backend.URL)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Переопределяем поведение director для кастомизации запроса
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Устанавливаем Host для backend-сервера
		req.Host = targetURL.Host

		// Прокидываем X-Real-IP
		req.Header.Set("X-Real-IP", clientIP)

		// Прокидываем X-Forwarded-For (дополняем цепочку)
		prior := req.Header.Get("X-Forwarded-For")
		if prior != "" {
			req.Header.Set("X-Forwarded-For", prior+", "+clientIP)
		} else {
			req.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	// Обработка ошибок при проксировании
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		h.Logger.Warnf("proxy error: %v", err)

		// Обработка критичных ошибок
		if isCriticalError(err) {
			http.Error(w, "Service unavailable due to backend error", http.StatusServiceUnavailable)
		} else {
			http.Error(w, "Backend error", http.StatusBadGateway)
		}

		h.BackendPool.MarkBackendAlive(backend.URL, false)
	}

	h.Logger.Infof("proxy %s -> %s", clientIP, backend.URL)

	backend.IncConnections()
	defer backend.DecConnections()

	proxy.ServeHTTP(w, r)
}

// Функция для проверки критичности ошибки
func isCriticalError(err error) bool {
    return strings.Contains(err.Error(), "database") || strings.Contains(err.Error(), "network")
}

// getClientIP извлекает IP-адрес клиента из заголовков X-Real-IP, X-Forwarded-For или из RemoteAddr
func getClientIP(r *http.Request) string {
    if ip := r.Header.Get("X-Real-IP"); ip != "" {
        return strings.TrimSpace(ip)
    }

    if ips := r.Header.Get("X-Forwarded-For"); ips != "" {
        // Берём первый IP из списка (оригинальный клиент)
        return strings.TrimSpace(strings.Split(ips, ",")[0])
    }

    // Если ничего нет в заголовках, берем IP из соединения
    host, _, _ := net.SplitHostPort(r.RemoteAddr)
    return host
}

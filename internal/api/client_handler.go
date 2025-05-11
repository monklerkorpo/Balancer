package api

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mk/loadBalancer/internal/ratelimiter"
	"github.com/mk/loadBalancer/internal/storage"
	"go.uber.org/zap"
)

// ClientLimitRequest — структура для парсинга запроса на создание/обновление лимита клиента.
type ClientLimitRequest struct {
	ClientID   string `json:"client_id"`     // Идентификатор клиента
	Capacity   int    `json:"capacity"`      // Максимальное количество токенов
	RefillRate int    `json:"rate_per_sec"`  // Скорость пополнения токенов в секунду
}

// ClientHandler обрабатывает HTTP-запросы, связанные с лимитами клиентов.
type ClientHandler struct {
	Repo   storage.ClientRepository      // Репозиторий для хранения лимитов клиентов
	Limiter *ratelimiter.RateLimiter     // Rate Limiter для ограничения запросов
	Logger *zap.SugaredLogger            // Логгер для записи действий и ошибок
}

// NewClientHandler создает новый экземпляр ClientHandler.
func NewClientHandler(repo storage.ClientRepository, limiter *ratelimiter.RateLimiter, logger *zap.SugaredLogger) *ClientHandler {
	return &ClientHandler{Repo: repo, Limiter: limiter, Logger: logger}
}

// RegisterRoutes регистрирует маршруты HTTP API для управления клиентами.
func (handler *ClientHandler) RegisterRoutes(r *mux.Router) {
   
    r.HandleFunc("", handler.List).Methods("GET")
    r.HandleFunc("", handler.Create).Methods("POST")
    r.HandleFunc("/{id}", handler.Get).Methods("GET")
    r.HandleFunc("/{id}", handler.Update).Methods("PUT")
    r.HandleFunc("/{id}", handler.Delete).Methods("DELETE")
}

// Create создает нового клиента с заданным лимитом.
func (handler *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req ClientLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.Logger.Warnw("невалидный JSON", "error", err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ClientID == "" || req.Capacity <= 0 || req.RefillRate <= 0 {
		handler.Logger.Warnw("невалидные данные клиента", "client_id", req.ClientID, "capacity", req.Capacity, "rate_per_sec", req.RefillRate)
		http.Error(w, "invalid client data", http.StatusBadRequest)
		return
	}

	limit := storage.ClientLimit{
		ClientID:   req.ClientID,
		Capacity:   req.Capacity,
		RefillRate: req.RefillRate,
	}

	if err := handler.Repo.Create(limit); err != nil {
		handler.Logger.Warnw("ошибка при создании клиента", "client_id", req.ClientID, "error", err)
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	handler.Limiter.SetClientLimit(req.ClientID, ratelimiter.ClientLimit{
		Capacity:   req.Capacity,
		RefillRate: req.RefillRate,
	})

	handler.Logger.Infow("клиент создан", "client_id", req.ClientID)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(limit)
}

// List возвращает список всех клиентов.
func (handler *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	clientLimits, err := handler.Repo.List()
	if err != nil {
		handler.Logger.Errorw("ошибка при получении списка клиентов", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clientLimits)
}

// Get возвращает лимит конкретного клиента по его ID.
func (handler *ClientHandler) Get(w http.ResponseWriter, r *http.Request) {
	clientID := mux.Vars(r)["id"]
	handler.Logger.Debugw("выполнен запрос на получение клиента", "path", r.URL.Path, "client_id", clientID)

	limit, err := handler.Repo.Get(clientID)
	if err != nil {
		handler.Logger.Warnw("клиент не найден", "client_id", clientID)
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	handler.Logger.Infow("клиент найден", "client_id", clientID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(limit)
}

// Update обновляет лимит клиента по ID.
func (handler *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	clientID := mux.Vars(r)["id"]

	var req ClientLimitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ClientID != clientID {
		handler.Logger.Warnw("невалидный запрос на обновление", "client_id", clientID, "error", err)
		http.Error(w, "invalid JSON or id mismatch", http.StatusBadRequest)
		return
	}

	if req.Capacity <= 0 || req.RefillRate <= 0 {
		handler.Logger.Warnw("невалидные значения при обновлении", "client_id", req.ClientID, "capacity", req.Capacity, "rate_per_sec", req.RefillRate)
		http.Error(w, "invalid capacity or rate_per_sec", http.StatusBadRequest)
		return
	}

	newLimit := storage.ClientLimit{
		ClientID:   req.ClientID,
		Capacity:   req.Capacity,
		RefillRate: req.RefillRate,
	}

	if err := handler.Repo.Update(newLimit); err != nil {
		handler.Logger.Warnw("не удалось обновить клиента", "client_id", clientID, "error", err)
		http.Error(w, "client not found", http.StatusNotFound)
		return
	}

	handler.Limiter.SetClientLimit(clientID, ratelimiter.ClientLimit{
		Capacity:   req.Capacity,
		RefillRate: req.RefillRate,
	})

	handler.Logger.Infow("клиент обновлен", "client_id", clientID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(newLimit)
}

// Delete удаляет клиента и его лимиты.
func (handler *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	clientID := mux.Vars(r)["id"]

	// Проверяем наличие клиента
	if _, err := handler.Repo.Get(clientID); err != nil {
		if err == storage.ErrNotFound {
			handler.Logger.Warnw("клиент не найден для удаления", "client_id", clientID)
			http.Error(w, "client not found", http.StatusNotFound)
			return
		}
		handler.Logger.Errorw("ошибка при проверке клиента", "client_id", clientID, "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Удаляем клиента из репозитория
	if err := handler.Repo.Delete(clientID); err != nil {
		handler.Logger.Errorw("не удалось удалить клиента", "client_id", clientID, "error", err)
		http.Error(w, "failed to delete client", http.StatusInternalServerError)
		return
	}

	// Удаляем из rate limiter
	handler.Limiter.RemoveClient(clientID)

	handler.Logger.Infow("клиент успешно удален", "client_id", clientID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "success",
		"message":   "client deleted",
		"client_id": clientID,
	})
}

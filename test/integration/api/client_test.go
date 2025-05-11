package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/mk/loadBalancer/internal/api"
	"github.com/mk/loadBalancer/internal/ratelimiter"
	"github.com/mk/loadBalancer/internal/storage"
	"go.uber.org/zap"
)

func setupTestRouter(t *testing.T) *mux.Router {
	t.Helper()

	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	// Используем in-memory SQLite
	repo, err := storage.NewSQLiteClientRepo(":memory:")
	if err != nil {
		t.Fatalf("Не удалось инициализировать репозиторий: %v", err)
	}

	rl := ratelimiter.NewRateLimiter(10, 1, repo, sugar)

	handler := api.NewClientHandler(repo, rl, sugar)
	// Монтируем роуты под префикс /clients
	r := mux.NewRouter()
	s := r.PathPrefix("/clients").Subrouter()
	handler.RegisterRoutes(s)

	return r
}

func TestClientCRUDOperations(t *testing.T) {
	router := setupTestRouter(t)

	clientID := "test-client"
	clientData := `{
		"client_id": "` + clientID + `",
		"capacity": 5,
		"rate_per_sec": 1
	}`

	// Тест 1: Создание клиента
	t.Run("Create Client", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/clients", strings.NewReader(clientData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusCreated {
			t.Fatalf("Ожидался статус 201 Created, получен %d", resp.Code)
		}

		// Проверим тело
		var got storage.ClientLimit
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		if got.ClientID != clientID || got.Capacity != 5 || got.RefillRate != 1 {
			t.Errorf("Unexpected client in response: %+v", got)
		}
	})

	// Тест 2: Дублирующий ID
	t.Run("Create Duplicate Client", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/clients", strings.NewReader(clientData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusConflict {
			t.Errorf("Expected 409 Conflict, got %d", resp.Code)
		}
	})

	// Тест 3: Получение клиента
	t.Run("Get Client", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/clients/"+clientID, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected 200 OK, got %d", resp.Code)
		}
	})

	// Тест 4: Несуществующий клиент
	t.Run("Get Nonexistent Client", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/clients/nonexistent", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found, got %d", resp.Code)
		}
	})

	// Тест 5: Обновление клиента
	t.Run("Update Client", func(t *testing.T) {
		updateData := `{
			"client_id": "` + clientID + `",
			"capacity": 10,
			"rate_per_sec": 2
		}`
		req := httptest.NewRequest(http.MethodPut, "/clients/"+clientID, strings.NewReader(updateData))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK on update, got %d", resp.Code)
		}

		// Проверим обновленные данные
		reqGet := httptest.NewRequest(http.MethodGet, "/clients/"+clientID, nil)
		respGet := httptest.NewRecorder()
		router.ServeHTTP(respGet, reqGet)

		if respGet.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK on get after update, got %d", respGet.Code)
		}

		var updated storage.ClientLimit
		if err := json.NewDecoder(respGet.Body).Decode(&updated); err != nil {
			t.Fatalf("Failed to decode updated client: %v", err)
		}
		if updated.Capacity != 10 || updated.RefillRate != 2 {
			t.Errorf("Update did not apply, got %+v", updated)
		}
	})

	// Тест 6: Удаление клиента
	t.Run("Delete Client", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/clients/"+clientID, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Errorf("Expected 200 OK on delete, got %d", resp.Code)
		}
	})

	// Тест 7: Удаление несуществующего
	t.Run("Delete Nonexistent Client", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/clients/nonexistent", nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Errorf("Expected 404 Not Found on delete nonexistent, got %d", resp.Code)
		}
	})

	// Тест 8: Пустое тело при создании
	t.Run("Create Empty Body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/clients", nil)
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request on empty body, got %d", resp.Code)
		}
	})

	// Тест 9: Некорректные данные
	t.Run("Create Invalid Data", func(t *testing.T) {
		invalid := `{
			"client_id": "` + clientID + `",
			"capacity": -1,
			"rate_per_sec": 1
		}`
		req := httptest.NewRequest(http.MethodPost, "/clients", strings.NewReader(invalid))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 Bad Request on invalid data, got %d", resp.Code)
		}
	})
}
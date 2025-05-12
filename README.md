# Вопросы для разогрева
🔹Опишите самую интересную задачу в программировании, которую вам приходилось решать?
-Разрабатывал API для библиотеки книг на Go, где главной задачей была оптимизация работы с PostgreSQL — изначально сделал нормализованную структуру с отдельными таблицами авторов и книг, но из-за медленных запросов денормализовал данные, добавив имя автора прямо в таблицу книг. Для ускорения поиска реализовал полнотекстовый поиск через PostgreSQL и кеширование популярных запросов в Redis. Столкнулся с проблемой конкурентного обновления рейтингов книг, которую решил с помощью транзакций и мьютексов. В процессе научился анализировать медленные запросы через EXPLAIN и правильно использовать индексы. Этот проект дал мне понимание, когда можно жертвовать идеальной структурой БД ради производительности.

🔹Расскажите о своем самом большом факапе?
-На хакатоне делал чат на WebSockets (Go + gorilla/websocket), но забыл закрывать соединения при выходе пользователей. Через час работы сервер съел всю память и упал — оказалось, горутины висели в фоне и копили данные. Пришлось срочно чинить на живом проекте перед демонстрацией жюри.

🔹Что вы предприняли для решения проблемы?
-Добавил defer conn.Close() и таймауты через context.WithTimeout, плюс настроил мониторинг утечек через pprof. Теперь перед каждым релизом запускаю проверку: go func() { for { log.Println("Num goroutines:", runtime.NumGoroutine()); time.Sleep(5 * time.Second) } }(). Этот опыт научил меня внимательнее работать с ресурсами в Go.

Каковы ваши ожидания от участия в буткемпе?
-Хочу глубже разобрать продвинутые темы Go: профилирование (pprof, trace), паттерны конкурентности (worker pools, graceful shutdown) и оптимизацию SQL-запросов. Особенно ценна возможность работать над реальными задачами в команде и получать фидбек от опытных разработчиков. Готов активно участвовать и применять знания на практике!

# ⚖️ Load Balancer with Rate Limiting (Go)

Проект реализует HTTP-прокси с балансировкой нагрузки, ограничением скорости запросов (rate limiting), health check'ами, а также возможностью управлять лимитами клиентов через REST API. Архитектура — **гексагональная**, модульная и масштабируемая.

##  Возможности

- ✅ **Round-Robin, least connections, random балансировка** между backend-серверами
- ✅ **Token Bucket Rate Limiter** (глобальный и индивидуальный per-client)
- ✅ **Health Check backend-ов** (исключение из пула при падении)
- ✅ **CRUD API** для управления лимитами клиентов (`/clients`)
- ✅ **Интеграция с Gorilla Mux**
- ✅ **Логирование** через `zap`
- ✅ **Мок-серверы** для тестов backend-сервисов
- ✅ **Бд**  Sqlite

##  Архитектура проекта

```
.
├── cmd/                # Главный исполняемый файл
├── internal/
│   ├── api/            # HTTP-обработчики и роутинг
│   ├── app/            # Точка входа и управление жизненным циклом сервера.
│   ├── server/         # Основной HTTP-сервер приложения, объединяющий все компоненты системы.
│   ├── balancer/       # Strategy, Backend, Health Check
│   ├── proxy/          # Прокси логика
│   ├── ratelimiter/    # Реализация Token Bucket Rate Limiter
│   └── storage/        # Sqlite реализация ClientRepository
├── test/               # Моковые backend-серверы // # Интеграционные тесты
├── configs/            # YAML конфиги
├── Dockerfile
├── docker-compose.yml
└── README.md           # Вы здесь!
```
##  Конфигурация 
```
port: 8080
backends:
  - "http://backend1:9001"  
  - "http://backend2:9002"
rate_limit:
  capacity: 100
  refill_rate: 10
databasePath: "clients.db"
strategy: round_robin  # можно заменить на least_connections , round_robin, random, потому что у нас есть фабрика стратегий.
```
##  Быстрый старт через Docker

```bash
docker-compose up --build
```

Порты:

- `:8080` — основной Load Balancer
- `:9001`, `:9002` — мок-сервера

##  HTTP API

###  `POST /clients`

Создание клиента:

```json
{
  "client_id": "user123",
  "capacity": 10,
  "rate_per_sec": 5
}
```

- `201 Created` — при успешном создании
- `409 Conflict` — клиент уже существует

###  `GET /clients`

Получить список всех клиентов.

###  `GET /clients/{id}`

Получить лимит клиента по ID.

###  `PUT /clients/{id}`

Обновить лимит клиента.

```json
{
  "client_id": "user123",
  "capacity": 15,
  "rate_per_sec": 10
}
```

- `400 Bad Request` — ID в URL и в теле не совпадают
- `404 Not Found` — клиент не найден

###  `DELETE /clients/{id}`

Удалить клиента.

- `200 OK` — клиент удалён
- `404 Not Found` — клиент не найден

##  Rate Limiting
Реализация Rate Limiting

🔹Система работает на основе Token Bucket алгоритма:
Для каждого клиента создается отдельный "ведро" токенов
Каждый запрос расходует один токен
Автопополнение токенов происходит через фиксированные интервалы (time.Ticker)

🔹Идентификация клиентов:
Приоритетно по заголовку X-API-Key
При отсутствии - по X-Real-IP/X-Forwarded-For
В крайнем случае - по RemoteAddr
При превышении лимита:
Возвращается статус 429 (Too Many Requests)
Добавляется заголовок Retry-After с временем ожидания

🔹Гибкость настроек:
Индивидуальные лимиты задаются через SetClientLimit(clientID, ClientLimit{...})
Возможна интеграция с внешними хранилищами (БД, Redis) для динамической загрузки правил




### 🔹 Глобальный лимит (по умолчанию):

- Конфигурируется в `ratelimiter.RateLimiter`
- Все клиенты подпадают под этот лимит, если не указан персональный

### 🔹 Индивидуальный лимит:

- Настраивается через API `/clients`
- Приоритетнее глобального

##  Health Checks

- Нездоровые сервера исключаются из пула

##  Интеграционные тесты

## Benchmark

Интеграционные бенчмарки для Rate Limiter запускались с флагами `-race` и `-benchmem`, чтобы оценить как производительность, так и потокобезопасность реализации.

Команда для запуска:

```bash
go test -bench=. -tags=integration -benchmem -race ./test/integration/ratelimiter
```

Покрывает:

- Ограничения по IP  
- Ограничения по API ключу  
- Индивидуальные лимиты на клиента  


##  Мок-серверы

Для имитации backend'ов:

- `test/mock/server1` — отвечает на все запросы с `Hello from mock server 1`
- `test/mock/server2` — аналогично, но с `mock server 2`



## Интеграционный тест CRUD в различных сценариях (подробный)
```
go test -v -count=1 ./test/integration/api
```

##  Пример запроса через Postman или curl

```bash
curl -X POST http://localhost:8080/clients \
  -H "Content-Type: application/json" \
  -d '{"client_id": "user1", "capacity": 10, "rate_per_sec": 5}'
```
## Пример тестирования ratelimiter определенного юзера
``` 
curl -X POST http://localhost:8080/clients      -H "Content-Type: application/json"      -d '{"client_id":"user4","capacity":4,"rate_per_sec":10}'
{"client_id":"user4","capacity":4,"rate_per_sec":10}
for i in {1..10}; do   curl -i http://localhost:8080/     -H "X-Client-ID: user4"; done

```
## Пример тестирования (ab -n 1000 -c 100 -H "X-Real-IP: 1.1.1.1" http://localhost:8080/) при общем rate_limit:  capacity: 100 refill_rate: 10
```
Concurrency Level:      100
Time taken for tests:   0.400 seconds
Complete requests:      1000
Failed requests:        900
   (Connect: 0, Receive: 0, Length: 900, Exceptions: 0)
Non-2xx responses:      900
Total transferred:      165400 bytes
HTML transferred:       43000 bytes
Requests per second:    2501.81 [#/sec] (mean)
Time per request:       39.971 [ms] (mean)
Time per request:       0.400 [ms] (mean, across all concurrent requests)
Transfer rate:          404.10 [Kbytes/sec] received
```

## Так же есть unit-тесты балансировщика 
```
go test -v ./internal/balancer/...
```


##  Технологии

- Go 1.21+
- Gorilla Mux
- Uber Zap (логирование)
- Docker / Docker Compose
- Sqlite

# balancer

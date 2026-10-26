# Secunda — API управления задачами

REST API для управления задачами внутри команд: роли, история изменений,
комментарии, кеширование списка задач и SQL-аналитика.
Реализованы аутентификация, команды, задачи, история, комментарии, Redis-кеш
и SQL-отчёт по команде.

## Стек

| Компонент | Технология |
|---|---|
| Язык | Go 1.26 |
| HTTP | chi + строгий сервер, сгенерированный из OpenAPI |
| БД | MySQL 8.0, `sqlx`, ручной SQL |
| Миграции | goose (встроены в бинарник) |
| Кеш | Redis 7 |
| Аутентификация | JWT HS256, bcrypt |
| Контракт | OpenAPI 3.0.3 → oapi-codegen |
| Документация | Swagger UI из встроенной спеки |
| Запуск | Docker + Docker Compose |

## Архитектура

Однонаправленные зависимости, «луковица»:

```
cmd/api                  — сборка приложения (composition root)
internal/domain          — сущности, роли, доменные ошибки и политики
internal/usecase         — бизнес-логика и порты (интерфейсы)
internal/adapter/mysql   — реализация портов поверх MySQL
internal/adapter/redis   — реализация кеша поверх Redis
internal/delivery/http   — HTTP-адаптер: OpenAPI, хендлеры, middleware
internal/platform        — конфиг, JWT, bcrypt (инфраструктура)
internal/migrations      — SQL-миграции, встроенные через embed
api/openapi.yaml         — единственный источник контракта
```

Правила слоёв:

- в хендлерах нет бизнес-логики — только проверка наличия body, вызов use case
  и маппинг доменного результата в сгенерированные модели;
- доступ и роли проверяются в `usecase` и `domain`, а не в HTTP-слое;
- `usecase` не знает ни про HTTP, ни про MySQL, ни про Redis — только про порты;
- SQL живёт только в `adapter/mysql`, Redis-команды — только в `adapter/redis`;
- границы транзакций задаёт use case через порт `TransactionManager`,
  репозитории автоматически подхватывают транзакцию из контекста.

## Быстрый старт

```bash
cp .env.example .env
make up
```

- API: <http://localhost:8080/api/v1>
- Swagger UI: <http://localhost:8080/swagger/> (только вне `production`)
- Health: <http://localhost:8080/healthz>, readiness: `/readyz`

Полезные команды:

```bash
make logs          # логи приложения
make ps            # статус контейнеров
make down          # остановить
make test          # тесты
make openapigen    # перегенерировать код из api/openapi.yaml
```

Интеграционный тест SQL-отчёта использует реальный MySQL и намеренно не
запускается без отдельной БД. После `make up` его можно выполнить так:

```bash
TEST_MYSQL_DSN='secunda:secunda@tcp(127.0.0.1:3306)/secunda?parseTime=true&loc=UTC&charset=utf8mb4' go test ./internal/adapter/mysql -run TestTeamStatsRepositoryIntegration
```

Если стандартные порты заняты, задайте в `.env`:
`MYSQL_HOST_PORT`, `REDIS_HOST_PORT`, `APP_HOST_PORT`.

Для локального запуска без Docker-образа приложения (MySQL и Redis — в compose):

```bash
make up            # поднимет mysql/redis/app; app можно остановить
docker compose stop app
make run
```

`make run` сам подставляет DSN на `127.0.0.1:<MYSQL_HOST_PORT>`.

## Конфигурация

Все значения читаются из окружения; секреты в коде не хранятся.

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `APP_ENV` | `local` | Окружение; при `production` Swagger UI и `/swagger/doc.json` отключены |
| `HTTP_ADDR` | `:8080` | Адрес HTTP-сервера |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `DB_DSN` | собирается автоматически | DSN MySQL; compose и `make run` подставляют его сами. Для прямого `go run` задайте вручную, обязательно `parseTime=true&loc=UTC` |
| `DB_MAX_OPEN_CONNS` | `25` | Пул соединений |
| `DB_MAX_IDLE_CONNS` | `25` | Пул соединений |
| `DB_CONN_MAX_LIFETIME` | `5m` | Время жизни соединения |
| `JWT_SECRET` | — (обязательно, ≥32 символов) | Ключ подписи JWT |
| `JWT_TTL` | `24h` | Время жизни access-токена |
| `BCRYPT_COST` | `12` | Стоимость bcrypt |
| `REDIS_ADDR` | `localhost:6379` | Redis |
| `REDIS_PASSWORD` | — | Пароль Redis |
| `REDIS_DB` | `0` | Номер БД Redis |
| `CACHE_TTL` | `5m` | TTL кеша списка задач |
| `AUTO_MIGRATE` | `true` | Применять миграции при старте |
| `SHUTDOWN_TIMEOUT` | `10s` | Graceful shutdown |

## Контракт и генерация

`api/openapi.yaml` — источник истины. Из него генерируются модели, chi-роутер
и строгий интерфейс сервера:

```bash
make openapigen
```

Конфигурация генерации — `api/oapi-codegen.yaml`, результат —
`internal/delivery/http/openapi/api.gen.go` (файл коммитится).
Swagger UI отдаёт ту же встроенную спеку по `/swagger/doc.json` и доступен
по `/swagger/` только вне `production`; в production маршруты `/swagger/*`
не регистрируются.

## Реализованные эндпоинты

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `POST` | `/api/v1/register` | публичный | Регистрация, выдача JWT |
| `POST` | `/api/v1/login` | публичный | Аутентификация, выдача JWT |
| `POST` | `/api/v1/teams` | JWT | Создание команды, автор становится owner |
| `GET` | `/api/v1/teams` | JWT | Только свои команды |
| `POST` | `/api/v1/teams/{id}/invite` | owner/admin | Приглашение по `user_id` и роли |
| `GET` | `/api/v1/teams/{team_id}/stats` | owner/admin | SQL-отчёт по одной команде |
| `POST` | `/api/v1/tasks` | участник команды | Создание задачи |
| `GET` | `/api/v1/tasks` | участник команды | Список с фильтрами и пагинацией |
| `PUT` | `/api/v1/tasks/{id}` | по правилам задачи | Обновление с optimistic locking |
| `GET` | `/api/v1/tasks/{id}/history` | участник команды | История изменений |
| `POST` | `/api/v1/tasks/{id}/comments` | участник команды | Добавление комментария |
| `GET` | `/api/v1/tasks/{id}/comments` | участник команды | Комментарии с пагинацией |

Правила приглашений:

- owner приглашает с ролью `admin` или `member`;
- admin приглашает только `member`;
- роль `owner` через приглашение не выдаётся;
- не участник команды получает `404` (существование чужой команды не
  раскрывается), участник без прав — `403`, несуществующий пользователь — `404`,
  повторное приглашение — `409`.

## Задачи: права и конкурентность

| Кто | Что может |
|---|---|
| owner / admin команды | редактировать любые поля любой задачи |
| автор задачи (любая роль) | редактировать заголовок, описание, статус, исполнителя |
| исполнитель | менять только статус (и производный `closed_at`) |
| остальные участники | читать задачи, историю и комментарии, писать комментарии |
| не участник команды | получает `404`, задача не раскрывается |

- Исполнителем может быть только участник той же команды — иначе `400`.
- `closed_at` выставляется при переходе в `done` и очищается при выходе из него.
- `PUT /tasks/{id}` требует поле `version`. Обновление выполняется в транзакции:
  `SELECT ... FOR UPDATE` → проверка прав и версии → `UPDATE ... WHERE version = ?`
  → запись истории. Устаревшая версия даёт `409 version_conflict`, а не потерю
  изменений. `assignee_id: 0` снимает исполнителя.
- История и изменение задачи пишутся в одной транзакции: если запись истории
  не удалась, задача не обновится.
- История хранит JSON-массив диффов `[{"field","old","new"}]`; при создании
  задачи фиксируется начальный снимок полей.

## Кеш списка задач

- Ключ: `secunda:tasks:v{version}:team:{team_id}:{sha256(filters)}`.
- Версия команды: `secunda:tasks:ver:{team_id}`. Создание и обновление задачи
  делают `INCR` — старые ключи становятся недостижимы без `SCAN`/`KEYS`.
- Версия читается **до** запроса в БД и используется при записи результата,
  поэтому гонка «инвалидация между чтением и записью» не приводит к stale-кешу.
- TTL списка — `CACHE_TTL` (по умолчанию `5m`).
- При недоступном Redis запросы обслуживаются из MySQL; ошибки кеша пишутся
  в лог и не ломают API. После аварии Redis возможна stale-выдача не дольше TTL.

### Примеры

```bash
# регистрация
curl -s -X POST http://localhost:8080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"ivan@example.com","password":"secret123","name":"Иван"}'

# логин
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ivan@example.com","password":"secret123"}' | jq -r .token)

# команда
curl -s -X POST http://localhost:8080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Платформа"}'

# приглашение
curl -s -X POST http://localhost:8080/api/v1/teams/1/invite \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"user_id":2,"role":"member"}'

# задача
TASK=$(curl -s -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"team_id":1,"title":"Подготовить релиз","assignee_id":2}')

# список с фильтром
curl -s "http://localhost:8080/api/v1/tasks?team_id=1&status=todo&limit=20&offset=0" \
  -H "Authorization: Bearer $TOKEN"

# обновление: version обязателен
curl -s -X PUT http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"status":"done","version":1}'

# история и комментарии
curl -s http://localhost:8080/api/v1/tasks/1/history -H "Authorization: Bearer $TOKEN"
curl -s -X POST http://localhost:8080/api/v1/tasks/1/comments \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"content":"Проверил миграции"}'
```

### Формат ошибок

```json
{
  "error": {
    "code": "forbidden",
    "message": "недостаточно прав",
    "request_id": "4f8c2f1b9a3e4d5f"
  }
}
```

`request_id` совпадает с заголовком `X-Request-Id` и попадает в структурные логи.
Ошибки валидации дополнительно содержат `details` с разбивкой по полям.

## Миграции

SQL-файлы лежат в `internal/migrations`, встроены в бинарник и применяются
goose при старте (`AUTO_MIGRATE=true`). Новая миграция — файл с префиксом
`NNNNN_`, например `00002_tasks.sql`, с блоками `-- +goose Up` / `-- +goose Down`.

## Надёжность

- создание команды и добавление owner выполняются в одной транзакции;
- обновление задачи и запись истории выполняются в одной транзакции;
- optimistic locking по `version` + `SELECT ... FOR UPDATE` исключает потерю
  изменений при конкурентном обновлении;
- кеш инвалидируется версией, без `SCAN`/`KEYS`; при отказе Redis API
  продолжает работать через MySQL;
- тело API-запроса ограничено 1 MiB;
- список публичных операций вычисляется из OpenAPI-спеки, а не задаётся руками;
- пул MySQL настроен, чтение конфигурации — из окружения;
- graceful shutdown по `SIGINT`/`SIGTERM`;
- структурированные JSON-логи (`slog`) с `request_id`;
- health/readiness-эндпоинты;
- Swagger UI и спека всегда соответствуют сгенерированному коду.

## Тесты

```bash
make test
```

Unit-тестами покрыты доменные правила ролей, права на поля задачи,
конкурентное обновление (bump версии, `409`), маппинг ошибок, JWT и
middleware. Интеграционный тест SQL-отчёта на реальном MySQL — этап 3.

## Этапы

- [x] **Этап 1.** Каркас onion-архитектуры, OpenAPI + генерация, MySQL,
  миграции, JWT, регистрация/логин, команды и приглашения, Swagger, Docker.
- [x] **Этап 2.** Задачи, история изменений, комментарии, optimistic locking,
  Redis-кеш списка задач с инвалидацией.
- [ ] **Этап 3.** SQL-отчёт `/teams/{team_id}/stats`, интеграционный тест
  отчёта, README по всем сценариям.

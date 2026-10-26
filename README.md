# Secunda — API управления задачами

REST API для управления задачами внутри команд: роли, история изменений,
комментарии, кеширование списка задач и SQL-аналитика.
Сейчас реализован **этап 1** — аутентификация, команды и приглашения.
Задачи, история, комментарии, Redis-кеш и отчёт — этапы 2–3 (см. ниже).

## Стек

| Компонент | Технология |
|---|---|
| Язык | Go 1.26 |
| HTTP | chi + строгий сервер, сгенерированный из OpenAPI |
| БД | MySQL 8.0, `sqlx`, ручной SQL |
| Миграции | goose (встроены в бинарник) |
| Кеш | Redis 7 (этап 2) |
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
internal/delivery/http   — HTTP-адаптер: OpenAPI, хендлеры, middleware
internal/platform        — конфиг, JWT, bcrypt (инфраструктура)
internal/migrations      — SQL-миграции, встроенные через embed
api/openapi.yaml         — единственный источник контракта
```

Правила слоёв:

- в хендлерах нет бизнес-логики — только проверка наличия body, вызов use case
  и маппинг доменного результата в сгенерированные модели;
- доступ и роли проверяются в `usecase` и `domain`, а не в HTTP-слое;
- `usecase` не знает ни про HTTP, ни про MySQL — только про порты;
- SQL живёт только в `adapter/mysql`.

## Быстрый старт

```bash
cp .env.example .env
make up
```

- API: <http://localhost:8080/api/v1>
- Swagger UI: <http://localhost:8080/swagger/index.html>
- Health: <http://localhost:8080/healthz>, readiness: `/readyz`

Полезные команды:

```bash
make logs          # логи приложения
make ps            # статус контейнеров
make down          # остановить
make test          # тесты
make openapigen    # перегенерировать код из api/openapi.yaml
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
| `APP_ENV` | `local` | Окружение |
| `HTTP_ADDR` | `:8080` | Адрес HTTP-сервера |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `DB_DSN` | собирается автоматически | DSN MySQL; compose и `make run` подставляют его сами. Для прямого `go run` задайте вручную, обязательно `parseTime=true&loc=UTC` |
| `DB_MAX_OPEN_CONNS` | `25` | Пул соединений |
| `DB_MAX_IDLE_CONNS` | `25` | Пул соединений |
| `DB_CONN_MAX_LIFETIME` | `5m` | Время жизни соединения |
| `JWT_SECRET` | — (обязательно, ≥32 символов) | Ключ подписи JWT |
| `JWT_TTL` | `24h` | Время жизни access-токена |
| `BCRYPT_COST` | `12` | Стоимость bcrypt |
| `REDIS_ADDR` | `localhost:6379` | Redis (этап 2) |
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
Swagger UI отдаёт ту же встроенную спеку по `/swagger/doc.json`.

## Реализованные эндпоинты (этап 1)

| Метод | Путь | Доступ | Описание |
|---|---|---|---|
| `POST` | `/api/v1/register` | публичный | Регистрация, выдача JWT |
| `POST` | `/api/v1/login` | публичный | Аутентификация, выдача JWT |
| `POST` | `/api/v1/teams` | JWT | Создание команды, автор становится owner |
| `GET` | `/api/v1/teams` | JWT | Только свои команды |
| `POST` | `/api/v1/teams/{id}/invite` | owner/admin | Приглашение по `user_id` и роли |

Правила приглашений:

- owner приглашает с ролью `admin` или `member`;
- admin приглашает только `member`;
- роль `owner` через приглашение не выдаётся;
- не участник команды получает `404` (существование чужой команды не
  раскрывается), участник без прав — `403`, несуществующий пользователь — `404`,
  повторное приглашение — `409`.

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
- тело API-запроса ограничено 1 MiB;
- список публичных операций вычисляется из OpenAPI-спеки, а не задаётся руками;
- пул MySQL настроен, чтение конфигурации — из окружения;
- graceful shutdown по `SIGINT`/`SIGTERM`;
- структурированные JSON-логи (`slog`) с `request_id`;
- health/readiness-эндпоинты;
- Swagger UI и спека всегда соответствуют сгенерированному коду.

## Этапы

- [x] **Этап 1.** Каркас onion-архитектуры, OpenAPI + генерация, MySQL,
  миграции, JWT, регистрация/логин, команды и приглашения, Swagger, Docker.
- [ ] **Этап 2.** Задачи, история изменений, комментарии, optimistic locking,
  Redis-кеш списка задач с инвалидацией.
- [ ] **Этап 3.** SQL-отчёт `/teams/{team_id}/stats`, интеграционный тест
  отчёта, unit-тесты прав, README по всем сценариям.

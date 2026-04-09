# Task Service

Сервис для управления задачами с HTTP API на Go.
Поддерживает обычные задачи и **шаблоны периодических задач** с автоматической генерацией экземпляров по расписанию.

## Требования

- Go `1.23+`
- Docker и Docker Compose

## Быстрый запуск через Docker Compose

```bash
docker compose up --build
```

После запуска сервис будет доступен по адресу `http://localhost:8080`.

Если `postgres` уже запускался ранее со старой схемой, пересоздай volume:

```bash
docker compose down -v
docker compose up --build
```

Причина в том, что SQL-файл из `migrations/0001_create_tasks.up.sql` монтируется в `docker-entrypoint-initdb.d` и применяется только при инициализации пустого data volume.

## Swagger

Swagger UI:

```text
http://localhost:8080/swagger/
```

OpenAPI JSON:

```text
http://localhost:8080/swagger/openapi.json
```

## API

Базовый префикс API:

```text
/api/v1
```

Основные маршруты:

- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{id}`
- `PUT /api/v1/tasks/{id}`
- `DELETE /api/v1/tasks/{id}`

- `POST /api/v1/templates`        Создать шаблон (сразу генерирует задачи на месяц вперёд) 
- `GET /api/v1/templates`         Получить список всех шаблонов 
- `GET /api/v1/templates/{id}`    Получить шаблон по ID 
- `PUT /api/v1/templates/{id}`    Обновить шаблон (удаляет будущие `new`-задачи и перегенерирует) 
- `DELETE /api/v1/templates/{id}` Удалить шаблон и все связанные экземпляры задач 

### Формат правил периодичности

- **daily** – `{ "interval_days": N }` (N >= 1)
- **monthly** – `{ "day_of_month": D }` (D от 1 до 31)
- **specific_dates** – `{ "dates": ["2025-05-01", "2025-06-12"] }`
- **even_odd** – `{ "parity": "even" }` или `"odd"`

### Автоматическая генерация задач

При создании шаблона

1. Генерируются задачи на **30 дней вперёд** от текущей даты
2. Для каждой даты проверяется соответствие правилу периодичности
3. Создаются только задачи, подходящие под правило

### Ограничения и валидация
Максимальный горизонт генерации — 1 месяц
Задачи на прошедшие даты не создаются
Для monthly с day_of_month = 31 — в месяцах с 30 днями задача не создаётся
Все даты и время обрабатываются в **Europe/Moscow** (МСК)

### Примеры работы правил

```json
// Каждые 3 дня
{ "interval_days": 3 }

// Каждое 15-е число месяца
{ "day_of_month": 15 }

// Только в указанные даты
{ "dates": ["2026-12-31", "2026-05-10"] }

// Только чётные дни месяца
{ "parity": "even" }
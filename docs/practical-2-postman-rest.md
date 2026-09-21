# Практическая работа № 2. Тестирование и анализ REST и RESTful API с использованием Postman

## Тема и цель

**Тема:** функциональное тестирование собственного REST API RepoScope в Postman.  
**Цель:** научиться формировать запросы с path/query-параметрами, заголовками и JSON-телом, анализировать коды и структуру ответов, автоматизировать проверки и выполнять связанный CRUD-сценарий.

## Инструменты

Postman Desktop или Web + Desktop Agent, Collection Runner, RepoScope API, PostgreSQL, GitHub REST API, JSON, OpenAPI 3. Файлы работы:

- `postman/RepoScope.postman_collection.json`;
- `postman/RepoScope.postman_environment.json`;
- `docs/openapi.yaml`.

## Теория: REST и RESTful

REST — архитектурный стиль для распределённых систем. Его ключевые идеи: разделение клиента и сервера, stateless-запросы, кэшируемость, единообразный интерфейс, слои и адресуемые ресурсы. RESTful API применяет эти идеи на практике: URL выражает ресурс, метод — намерение, HTTP status — результат, представление обычно передаётся JSON.

RepoScope использует существительные `/repositories` и `/favorites`; чтение выполняет `GET`, создание элемента коллекции — `POST`, удаление адресуемого элемента — `DELETE`. Сервер не хранит HTTP-сессию. `201` содержит `Location`, `204` не содержит body. Это ближе к REST, чем RPC-маршруты вида `/addFavorite`.

## Постановка задачи

Импортировать готовые артефакты, запустить коллекцию на локальном стенде и проверить:

- положительные `200`, `201`, `204`;
- отрицательные `400`, `404`, `409`, `422`;
- `Content-Type: application/json` для ответов с телом;
- объект/массивы и обязательные JSON-поля;
- path и query parameters, пользовательский `X-Request-ID`;
- сохранение `requestId`, `favoriteLocation` и данных поиска;
- последовательное добавление, конфликт и удаление избранного;
- корректную передачу лимита GitHub как `429` (если лимит реально исчерпан).

## Импорт и запуск

1. Запустить стенд: `docker compose up --build -d`.
2. В Postman выбрать **Import → Files** и импортировать оба JSON из каталога `postman`.
3. Активировать environment **RepoScope Local**.
4. При необходимости изменить `baseUrl`, `owner`, `repo`, `query`, `language`.
5. Открыть Collection Runner, выбрать **RepoScope API**, включить сохранение environment и выполнить один iteration строго в порядке коллекции.
6. Изучить вкладки Test Results, Console и заголовок `X-Request-ID`.

Коллекция специально начинает lifecycle избранного с pre-clean. Поэтому результат не зависит от наличия записи после предыдущего запуска. Pre-clean принимает `204` или `404`, создание затем обязано вернуть `201`.

## Переменные

| Переменная | Пример | Назначение |
|---|---|---|
| `baseUrl` | `http://localhost:8080/api/v1` | База собственного API |
| `owner` | `golang` | Path и POST body |
| `repo` | `go` | Path и POST body |
| `query` | `http server` | Поисковая строка |
| `language` | `Go` | Фильтр GitHub |
| `page`, `perPage` | `1`, `10` | Пагинация |
| `favoriteLocation` | заполняется тестом | Проверка `Location` |
| `requestId` | заполняется тестом | Корреляция с логом |
| `sampleOwner`, `sampleRepo` | заполняются поиском | Пример collection scope |

## Автоматические тесты

На уровне коллекции для каждого ответа с телом проверяется JSON Content-Type, сохраняется `X-Request-ID`, контролируется время менее 10 секунд. Пример скрипта запроса:

```javascript
pm.test('Status 201', () => pm.response.to.have.status(201));
pm.test('Location present', () =>
  pm.expect(pm.response.headers.get('Location'))
    .to.match(/^\/api\/v1\/favorites\//));
const json = pm.response.json();
pm.expect(json).to.include.keys('id', 'owner', 'repo', 'created_at');
pm.environment.set('favoriteLocation', pm.response.headers.get('Location'));
```

Тест issues перебирает элементы и подтверждает отсутствие `pull_request`. Поиск проверяет `items`, `page`, `total_count`. Детали проверяют диапазон `health_score` от 0 до 100.

## Примеры запросов и ожидаемых ответов

### Создание

```http
POST /api/v1/favorites HTTP/1.1
Content-Type: application/json
Accept: application/json

{"owner":"golang","repo":"go"}
```

Ожидается `201 Created`, `Location: /api/v1/favorites/golang/go` и объект:

```json
{
  "id": 1,
  "owner": "golang",
  "repo": "go",
  "full_name": "golang/go",
  "description": "The Go programming language",
  "html_url": "https://github.com/golang/go",
  "stars": 0,
  "language": "Go",
  "created_at": "2026-09-20T10:00:00Z"
}
```

`id`, время, описание и stars зависят от базы и актуального GitHub. Повтор того же POST ожидаемо даёт:

```json
{
  "error": {
    "code": "favorite_exists",
    "message": "Репозиторий уже добавлен в избранное",
    "request_id": "...",
    "details": null
  }
}
```

со статусом `409 Conflict`.

### Валидация

`GET /repositories/search?q=x&page=0` возвращает `422`, поскольку проверка `q` срабатывает первой. Синтаксически оборванное тело `{"owner":` возвращает `400 invalid_json`. Отсутствующий GitHub-репозиторий возвращает `404 github_not_found`.

## Таблица тестового покрытия

| Сценарий | Метод и URL | Параметры/body | Ожидается | Проверка |
|---|---|---|---|---|
| Liveness | GET `/health` | Accept | 200 | `status=ok` |
| Readiness | GET `/ready` | — | 200 | `status=ready` |
| Rate limit | GET `/rate-limit` | — | 200 | `core`, `search` |
| Search | GET `/repositories/search` | query + header | 200 | page/items/total |
| Details | GET `/repositories/{owner}/{repo}` | path | 200 | owner, score |
| Child resources | GET `…/languages|contributors|issues|releases` | path/query | 200 | структуры |
| Bad query | GET `/repositories/search?q=x` | query | 422 | единая ошибка |
| Missing | GET `/repositories/.../missing` | path | 404 | error code |
| Create | POST `/favorites` | JSON | 201 | Location + DTO |
| Duplicate | POST `/favorites` | тот же JSON | 409 | `favorite_exists` |
| List | GET `/favorites` | — | 200 | созданная запись |
| Delete | DELETE `/favorites/{owner}/{repo}` | path | 204 | пустое тело |
| Repeat delete | DELETE тот же URL | path | 404 | `favorite_not_found` |
| Malformed body | POST `/favorites` | битый JSON | 400 | `invalid_json` |

## Анализ REST-соответствия

Сильные стороны: resource-oriented URL, правильная семантика методов и кодов, единый media type, stateless API, `Location`, явная пагинация, корреляционный заголовок. GET upstream кэшируется внутри сервиса; наружные cache headers пока не публикуются. HATEOAS-ссылки между ресурсами отсутствуют, но REST не требует максимального Richardson Maturity Level для учебного API. Избранное пока single-tenant: появление пользователей потребует ресурса `/users/{id}/favorites` или определения текущего пользователя через аутентификацию.

## Обработка ошибок

Postman не считает любой HTTP-ответ сетевой ошибкой: статус проверяется явно. Для `429` ручной анализ должен учитывать `details.retry_after`; коллекция не пытается искусственно исчерпать лимит, поскольку это вредно и неповторяемо. Ошибка readiness `503` означает проблему БД, `502` — проблему upstream, что помогает локализовать сбой.

## Вывод

Создана импортируемая коллекция с environment, позитивными и негативными тестами, переменными и зависимым lifecycle. Проверены методы, URL, параметры, заголовки, тела, коды и JSON-контракты. Анализ показывает, что API RepoScope практически следует принципам REST и даёт предсказуемые машинно-читаемые ошибки.


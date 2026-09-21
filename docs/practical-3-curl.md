# Практическая работа № 3. Использование cURL для выполнения запросов к REST API

## Тема и цель

**Тема:** выполнение и диагностика HTTP-запросов RepoScope из командной строки.  
**Цель:** освоить cURL для GET, POST и DELETE, передачи path/query-параметров, заголовков и JSON body, сохранения ответа, просмотра протокола и программной проверки HTTP-кода.

## Инструменты

cURL 8.x, Bash для Unix-подобных ОС, PowerShell и `curl.exe` для Windows, RepoScope API, JSON. Готовые сценарии: `scripts/curl-demo.sh` и `scripts/curl-demo.ps1`; краткая справка: `docs/curl-cheatsheet.md`.

## Краткая теория

cURL — CLI-клиент передачи данных по URL. Для HTTP запрос состоит из метода, target URL, заголовков и необязательного тела. cURL пишет body в stdout, диагностический verbose-поток — в stderr, а exit code сообщает о транспортной ошибке. Без `--fail` HTTP 404 сам по себе не даёт ненулевой exit code, поэтому автоматизация отдельно читает `%{http_code}`.

Path parameters являются сегментами `/repositories/{owner}/{repo}`. Query parameters идут после `?`; для пользовательского текста безопаснее `--get --data-urlencode`. Заголовок `Content-Type` описывает отправляемое тело, `Accept` — желаемый ответ.

## Постановка задачи

Выполнить системную проверку, поиск, карточку, добавление и удаление избранного; показать заголовки, verbose, сохранение JSON; проверить позитивные и негативные статусы. Windows-скрипт обязан вызывать `curl.exe`, исключая неоднозначность PowerShell alias.

## Подготовка и запуск

```bash
docker compose up --build -d
bash scripts/curl-demo.sh
```

PowerShell:

```powershell
docker compose up --build -d
.\scripts\curl-demo.ps1
```

Параметры можно заменить:

```bash
BASE_URL=http://localhost:8080/api/v1 OWNER=go-chi REPO=chi bash scripts/curl-demo.sh
```

```powershell
.\scripts\curl-demo.ps1 -Owner go-chi -Repo chi -OutFile result.json
```

## Порядок выполнения и команды

### GET, headers и query parameters

```bash
curl --fail-with-body --silent --show-error --include \
  -H 'Accept: application/json' \
  'http://localhost:8080/api/v1/health'

curl --silent --show-error --get \
  -H 'Accept: application/json' \
  -H 'X-Request-ID: curl-lab-search' \
  --data-urlencode 'q=http server' \
  --data-urlencode 'language=Go' \
  --data 'sort=stars' --data 'page=1' --data 'per_page=5' \
  'http://localhost:8080/api/v1/repositories/search'
```

Ожидается `200`, JSON с `items`, `total_count`, `page`, `per_page`, `total_pages`, а в ответе — `X-Request-ID`.

### Path parameters и verbose

```bash
curl --verbose --output /dev/null \
  'http://localhost:8080/api/v1/repositories/golang/go'
```

`--verbose` показывает строки запроса `>` и ответа `<`, соединение и TLS. Токен в таком выводе был бы виден, поэтому verbose-лог нельзя без проверки публиковать.

### JSON body и POST

```bash
curl --include --request POST \
  -H 'Content-Type: application/json' \
  -H 'Accept: application/json' \
  --data '{"owner":"golang","repo":"go"}' \
  'http://localhost:8080/api/v1/favorites'
```

После предварительной очистки ожидается `201 Created`, заголовок `Location` и JSON избранного. Повторный запрос — `409 Conflict`.

### DELETE

```bash
curl --include --request DELETE \
  'http://localhost:8080/api/v1/favorites/golang/go'
```

Первый ответ — `204 No Content` без body. Повторный — `404` с JSON-ошибкой.

### Только заголовки и сохранение body

Для API, не объявляющего отдельный HEAD, выполняется обычный GET, заголовки пишутся в stdout, а body отбрасывается:

```bash
curl --silent --dump-header - --output /dev/null \
  'http://localhost:8080/api/v1/health'
```

```bash
curl --silent --output response.json \
  'http://localhost:8080/api/v1/repositories/golang/go'
```

### Проверка статуса

```bash
code=$(curl --silent --output response.json --write-out '%{http_code}' \
  'http://localhost:8080/api/v1/repositories/golang/go')
if [ "$code" != 200 ]; then echo "Unexpected HTTP $code" >&2; exit 1; fi
```

PowerShell:

```powershell
$code = curl.exe --silent --output response.json --write-out "%{http_code}" `
  "http://localhost:8080/api/v1/repositories/golang/go"
if ([int]$code -ne 200) { throw "Unexpected HTTP $code" }
```

## Ожидаемые ответы

Health:

```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8
X-Request-Id: ...

{"service":"reposcope-api","status":"ok"}
```

Невалидный поиск `?q=x&page=0`:

```http
HTTP/1.1 422 Unprocessable Entity
Content-Type: application/json; charset=utf-8

{"error":{"code":"validation_error","message":"Параметр q должен содержать от 2 до 256 символов","request_id":"...","details":{"q":"invalid"}}}
```

Дубликат:

```http
HTTP/1.1 409 Conflict
Content-Type: application/json; charset=utf-8

{"error":{"code":"favorite_exists","message":"Репозиторий уже добавлен в избранное","request_id":"...","details":null}}
```

## Таблица запросов

| Метод | URL | Что передаётся | Ожидаемые коды |
|---|---|---|---|
| GET | `/health` | `Accept` | 200 |
| GET | `/repositories/search` | query, `X-Request-ID` | 200, 422, 429 |
| GET | `/repositories/{owner}/{repo}` | path | 200, 404, 429 |
| GET | `/rate-limit` | — | 200, 429, 502 |
| POST | `/favorites` | Content-Type + JSON body | 201, 400, 404, 409, 422, 429 |
| DELETE | `/favorites/{owner}/{repo}` | path | 204, 404, 422 |

## Анализ и обработка ошибок

Скрипты используют `-sS`, чтобы убрать прогресс, но сохранить ошибки; сохраняют body отдельно от `%{http_code}`; функция `check_code`/`Assert-Code` завершает сценарий при расхождении. `--fail-with-body` полезен для одиночных команд, но в негативном тесте 409/422 ожидаемы, поэтому статус сравнивается вручную. URL-кодирование предотвращает повреждение пробелов и специальных символов.

Транспортная ошибка (сервер не запущен) отличается от HTTP-ошибки: cURL вернёт ненулевой exit code и не выдаст нормальный status. `429` означает не повторять запрос немедленно; следует проверить `/rate-limit`, `Retry-After` или дождаться reset.

## Вывод

Освоены методы, path/query, заголовки, JSON, подробный режим, вывод заголовков, запись в файл и автоматическая проверка кодов. Два переносимых сценария позволяют повторить лабораторную работу в Bash и PowerShell, причём Windows-вариант явно использует `curl.exe`.

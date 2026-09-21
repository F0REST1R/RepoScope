# cURL: краткая памятка RepoScope

Базовый адрес: `http://localhost:8080/api/v1`. В PowerShell всегда используйте `curl.exe`: имя `curl` в старых версиях PowerShell могло быть псевдонимом `Invoke-WebRequest`.

## Основные флаги

| Флаг | Назначение |
|---|---|
| `-X`, `--request` | Явно задать метод (`POST`, `DELETE`) |
| `-H`, `--header` | Передать HTTP-заголовок |
| `-d`, `--data` | Передать тело; с `GET` удобнее применять вместе с `--get` |
| `--data-urlencode` | URL-кодировать query-параметр |
| `-i`, `--include` | Вывести заголовки вместе с телом |
| `-D - -o /dev/null` | Вывести заголовки обычного GET и отбросить body |
| `-v`, `--verbose` | Показать DNS, соединение, запрос и ответ; не публикуйте вывод с токенами |
| `-o file.json` | Сохранить тело в файл |
| `-w "%{http_code}"` | Вывести HTTP-код отдельно |
| `-sS` | Тихий режим, но с сообщениями об ошибках |
| `--fail-with-body` | Ненулевой exit code при 4xx/5xx, не скрывая тело |

## Примеры

```bash
# GET + query parameters
curl -sS --get -H 'Accept: application/json' \
  --data-urlencode 'q=web framework' --data-urlencode 'language=Go' \
  --data 'sort=stars' --data 'page=1' \
  'http://localhost:8080/api/v1/repositories/search'

# GET + path parameters
curl -sS 'http://localhost:8080/api/v1/repositories/golang/go'

# POST + заголовки + JSON body
curl -i -X POST -H 'Content-Type: application/json' \
  -d '{"owner":"golang","repo":"go"}' \
  'http://localhost:8080/api/v1/favorites'

# DELETE: успешный ответ 204 не имеет тела
curl -i -X DELETE 'http://localhost:8080/api/v1/favorites/golang/go'

# Только заголовки (GET; body отбрасывается)
curl -sS -D - -o /dev/null 'http://localhost:8080/api/v1/health'

# Сохранить тело и проверить код
code=$(curl -sS -o result.json -w '%{http_code}' \
  'http://localhost:8080/api/v1/repositories/golang/go')
test "$code" = 200
```

PowerShell:

```powershell
$code = curl.exe --silent --output result.json --write-out "%{http_code}" `
  "http://localhost:8080/api/v1/repositories/golang/go"
if ([int]$code -ne 200) { throw "HTTP $code" }
```

Полные повторяемые сценарии находятся в [`scripts/curl-demo.sh`](../scripts/curl-demo.sh) и [`scripts/curl-demo.ps1`](../scripts/curl-demo.ps1). Они включают позитивные запросы, `400/409/422`, сохранение ответа и проверку кода.

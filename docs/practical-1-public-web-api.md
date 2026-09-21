# Практическая работа № 1. Знакомство с публичными Web API

## Тема и цель

**Тема:** применение GitHub REST API в программной системе RepoScope.  
**Цель:** изучить назначение публичного Web API, выполнить запросы к нему и преобразовать внешние JSON-данные в полезное представление для поиска и оценки open-source проектов.

## Инструменты

Go 1.23.2, `net/http`, chi, GitHub REST API, JSON, React/TypeScript, Docker Compose, браузер. Официальные источники: [GitHub REST API](https://docs.github.com/en/rest), [версии API](https://docs.github.com/en/rest/about-the-rest-api/api-versions), [rate limits](https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api).

## Краткая теория

API (Application Programming Interface) — формальный способ взаимодействия программных компонентов. Web API предоставляет операции по HTTP. Клиент формирует метод, URL, заголовки и иногда тело; сервер возвращает статус, заголовки и представление ресурса. REST API обычно адресует ресурсы URL, использует семантику HTTP-методов и stateless-запросы. JSON — текстовый формат объектов, массивов, чисел, строк, логических и `null`-значений.

GitHub выбран потому, что предоставляет реальные, хорошо документированные публичные данные, знакомые разработчикам: репозитории, владельцев, темы, языки, участников, issues и релизы. Эти данные решают прикладную задачу предварительной оценки проекта до изучения кода или первого contribution.

## Характеристики внешнего API

- базовый URL: `https://api.github.com`;
- формат: JSON, `Accept: application/vnd.github+json`;
- версия: `X-GitHub-Api-Version: 2026-03-10` (переменная `GITHUB_API_VERSION`);
- идентификация клиента: `User-Agent: RepoScope/1.0`;
- авторизация: необязательный `Authorization: Bearer <token>` формируется только backend;
- публичный режим: 60 core-запросов в час с IP; типичный лимит personal access token — 5000 в час; search имеет отдельный bucket;
- при исчерпании лимита GitHub возвращает `403` или `429`; необходимо учитывать `Retry-After` и `X-RateLimit-Reset`.

Версия `2026-03-10` является текущей на дату работы; GitHub указывает версию датой релиза и поддерживает предыдущую как минимум ещё 24 месяца. Токен не обязателен для публичных ресурсов и никогда не передаётся в браузер.

## Постановка задачи

Создать backend-прокси, который принимает удобные запросы RepoScope, валидирует их, запрашивает публичные ресурсы GitHub, скрывает детали авторизации, нормализует ошибки, фильтрует pull requests из выдачи issues и рассчитывает понятный Health Score. UI должен поддерживать поиск, язык, сортировку, пагинацию и подробную карточку.

## Используемые внешние endpoints

| Назначение | Метод и GitHub URL | Параметры | Ожидаемый успех |
|---|---|---|---|
| Поиск | `GET /search/repositories` | `q`, `sort`, `order`, `page`, `per_page` | 200 |
| Репозиторий | `GET /repos/{owner}/{repo}` | path | 200, 404 |
| Языки | `GET /repos/{owner}/{repo}/languages` | path | 200 |
| Участники | `GET /repos/{owner}/{repo}/contributors` | `page`, `per_page` | 200 |
| Issues | `GET /repos/{owner}/{repo}/issues` | `state`, `sort`, `page`, `per_page` | 200 |
| Релизы | `GET /repos/{owner}/{repo}/releases` | `page`, `per_page` | 200 |
| Лимит | `GET /rate_limit` | — | 200 |

## Описание реализации

`internal/github.Client` создаёт запрос через `http.NewRequestWithContext`, поэтому закрытие браузерного соединения отменяет upstream-вызов. Клиент имеет timeout 12 секунд, ограничивает ответ 8 MiB и кэширует успешные GET на 2 минуты. Токен добавляется только если `GITHUB_TOKEN` непуст.

GitHub JSON декодируется в минимальные типизированные модели: лишние внешние поля сознательно игнорируются. RepoScope:

- объединяет `q` и квалификатор `language:<язык>`;
- переименовывает структуру пагинации в стабильный внутренний контракт;
- исключает элементы с `pull_request` из `/issues`;
- оборачивает карту языков в `{ "languages": ... }`;
- добавляет `health_score` и `health_summary` к карточке;
- переводит внешний 404 в собственный 404, а 403/429 лимита — в 429;
- не возвращает токен или сырые внутренние ошибки.

## Порядок выполнения

1. Запустить `docker compose up --build -d` и дождаться healthy-состояния.
2. Проверить `GET http://localhost:8080/api/v1/health`.
3. Выполнить поиск с `q=http server&language=Go&sort=stars`.
4. Выбрать `full_name`, запросить карточку и вложенные ресурсы.
5. Сравнить исходные поля GitHub с нормализованным ответом RepoScope.
6. Проверить `/rate-limit` сначала без токена, затем при необходимости с локальным `GITHUB_TOKEN` в `.env`.
7. Проверить несуществующий репозиторий и невалидный query.

## Примеры запросов и ожидаемых ответов

```http
GET /api/v1/repositories/search?q=http%20server&language=Go&sort=stars&order=desc&page=1&per_page=2 HTTP/1.1
Host: localhost:8080
Accept: application/json
```

Ожидаемая форма `200 OK` (счётчики меняются вместе с GitHub):

```json
{
  "items": [{
    "id": 123,
    "name": "project",
    "full_name": "owner/project",
    "description": "Example public repository",
    "html_url": "https://github.com/owner/project",
    "stargazers_count": 1000,
    "forks_count": 100,
    "open_issues_count": 20,
    "owner": {"login": "owner", "avatar_url": "https://...", "html_url": "https://github.com/owner"}
  }],
  "total_count": 42,
  "page": 1,
  "per_page": 2,
  "total_pages": 21
}
```

```http
GET /api/v1/repositories/golang/go/languages HTTP/1.1
```

```json
{"languages":{"Go":123456789,"Assembly":123456,"C":12345}}
```

Значения выше иллюстрируют реальную структуру ответа, но не выдаются за фиксированный снимок GitHub: популярность и состав языков изменяются.

## Таблица собственного API

| Метод | URL | Параметры | Коды |
|---|---|---|---|
| GET | `/repositories/search` | query `q`, `language`, `sort`, `order`, `page`, `per_page` | 200, 422, 429, 502 |
| GET | `/repositories/{owner}/{repo}` | path `owner`, `repo` | 200, 404, 422, 429 |
| GET | `…/languages` | path | 200, 404, 429 |
| GET | `…/contributors` | path + pagination | 200, 404, 422, 429 |
| GET | `…/issues` | path + pagination | 200, 404, 422, 429 |
| GET | `…/releases` | path + pagination | 200, 404, 422, 429 |
| GET | `/rate-limit` | — | 200, 429, 502 |

## Ошибки и анализ результатов

Для `q=x` ожидается `422` с `error.code=validation_error`. Для несуществующего репозитория — `404 github_not_found`. При лимите — `429 github_rate_limited` с возможным `details.retry_after`. Timeout/недоступность GitHub не маскируются под пустую выдачу: клиент получает контролируемую ошибку.

Архитектура решает проблему CORS/секретов, создаёт стабильный внутренний контракт и позволяет заменять GitHub mock-сервером только в автоматических тестах. Ограничение решения — изменяемость публичных данных и лимит первой тысячи результатов поиска GitHub.

## Вывод

В работе изучены URL, методы, заголовки, JSON, авторизация, версия и ограничения публичного Web API. Реализована прикладная система, которая не просто выводит ответ GitHub, а валидирует вход, преобразует данные, добавляет вычисляемый показатель и корректно сообщает об ошибках и лимитах.

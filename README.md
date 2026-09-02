# Rookery

Репозиторий: https://github.com/proxt/rookery
Нода (relay-сервер): https://github.com/proxt/rookery-node

WebRTC-туннель: произвольный TCP/UDP поверх WebRTC DataChannel между Windows-клиентом
и relay-нодой, так что на уровне сети сессия выглядит как обычный p2p-трафик.

Архитектура — панель + подписки + ноды, в духе Marzban/Remnawave: этот репозиторий
содержит **панель** (управление пользователями, подписками, статистикой, регистрация
нод) и **Windows-клиент**. Сами relay-ноды — отдельный компонент
([proxt/rookery-node](https://github.com/proxt/rookery-node)), который панель может
регистрировать и мониторить в любом количестве. Одна **подписка** пользователя может
включать сразу несколько нод — как в happ/v2ray.

## Структура репозитория

- `shared/` — код, общий для панели и клиента: формат заголовка адресата (`protocol`),
  токены сессий (`signaling`)
- `panel/` — панель (`cmd/rookeryp`): пользователи, подписки, регистрация нод,
  статистика (SQLite), веб-админка (`internal/admin`), эндпоинт `/sub/{token}` для
  клиента и `/api/nodes/*` для нод. Собирается `CGO_ENABLED=0` под linux/amd64,
  публикуется как Docker-образ
- `client/` — Windows-клиент: чистое ядро (`internal/engine`), headless CLI
  (`cmd/rookery-cli`) и Wails GUI (`gui/`)

Модули объединены через `go.work` в корне.

## Как это устроено

- **Пользователь** — просто владелец подписок.
- **Подписка** — токен вида `/sub/{token}`, за которым стоит набор нод. Клиент
  запрашивает этот URL и получает список доступных нод, каждая — с одноразовым
  подписанным токеном сессии (HMAC ключом самой ноды, панель никогда не даёт ноде
  проверять токен через сеть — верификация локальная).
- **Нода** не хранит пользователей вообще — только `node_id`/`api_key`, выданные
  при регистрации в панели. Периодически шлёт heartbeat и отчёт по трафику
  (`bytes_up`/`bytes_down` на подписку) обратно в панель.

Подробности протокола нода↔панель — в README [rookery-node](https://github.com/proxt/rookery-node).

## Сборка

Требуется Go 1.27+.

```
make build-panel   # bin/rookeryp — статический линукс-бинарь, CGO_ENABLED=0
make build-cli     # bin/rookery-cli — headless клиент
make build-gui     # GUI на Wails; см. раздел ниже про требования
make docker-build  # локальная сборка Docker-образа панели (для теста, без публикации)
```

```
make test   # go test по всем модулям
make lint   # go vet по всем модулям
make clean  # удалить bin/
```

### GUI (Wails)

`build-gui` требует Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
и **WebView2 Runtime**. Собирать нужно на Windows или в CI на `windows-latest` — кросс-компиляция
GUI с CGO-зависимостями (`fyne.io/systray`, `wailsapp/go-webview2`) с Linux не поддерживается.

## Установка панели на Ubuntu (Docker, рекомендуется)

При каждом пуше в `master`, затрагивающем `panel/` или `shared/`, GitHub Actions
собирает образ и публикует его в `ghcr.io/proxt/rookery-panel` (см.
`.github/workflows/panel-docker.yml`).

1. Поставить Docker: `curl -fsSL https://get.docker.com | sudo sh`
2. Скачать compose-файл:
   ```
   mkdir -p ~/rookery-panel && cd ~/rookery-panel
   curl -O https://raw.githubusercontent.com/proxt/rookery/master/panel/deploy/docker-compose.yml
   ```
3. Запустить:
   ```
   sudo docker compose up -d
   sudo docker compose logs
   ```
   В логах при первом запуске будет строка вида
   `generated admin panel credentials ... username=admin password=...` —
   сохраните пароль, второй раз он не покажется.
4. Поставить Caddy на хосте (не в контейнере) и взять за основу
   `panel/deploy/Caddyfile.example` — подставить свой домен. Caddy получит
   TLS-сертификат сам и проксирует на `127.0.0.1:8090`.
5. Зайти на `https://ваш-домен/admin`, авторизоваться, в настройках указать
   «Публичный адрес панели» (используется для формирования ссылок подписок).
6. На вкладке «Ноды» добавить relay-ноду (см. [rookery-node](https://github.com/proxt/rookery-node)
   для установки самой ноды) — панель выдаст `node_id`/`api_key`, которые
   идут в `node.yaml` той ноды.
7. Создать пользователя и подписку, привязать к ней ноды — панель даст
   ссылку подписки (`sub_url`).

Обновление: `sudo docker compose pull && sudo docker compose up -d`. Данные
(пользователи, подписки, ноды, логин администратора) лежат в volume
`rookery_panel_data`, а не в образе.

Пакет `ghcr.io/proxt/rookery-panel` публичный, `docker pull` на VDS работает
без `docker login`.

### Порты

- **TCP 443** (или 80 для ACME) — снаружи, на Caddy, проксирует на панель
  (`127.0.0.1:8090` по умолчанию).
- Панель не участвует в WebRTC/ICE напрямую — это чистый HTTP-сервис. UDP-порты
  нужны только нодам, см. их README.

## Клиент

**Статус:** клиент ещё не переведён на модель подписок — это следующий этап
работы. Сейчас `client/internal/engine` умеет установить сессию только по
готовому токену сессии (`SessionRequest.Token`), но GUI/CLI пока не знают,
как получить такой токен через `/sub/{token}` и выбрать ноду из списка.
До этого этапа клиент не подключится к панели/ноде из этого репозитория.

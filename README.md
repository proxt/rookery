# Rookery

Репозиторий: https://github.com/proxt/rookery

WebRTC-туннель: произвольный TCP/UDP поверх WebRTC DataChannel между Windows-клиентом
и нодой на Ubuntu, так что на уровне сети сессия выглядит как обычный p2p-трафик.

Статус: сквозной туннель работает — SOCKS5 CONNECT (TCP) и UDP ASSOCIATE проверены
end-to-end, реконнект с экспоненциальным backoff и backpressure по bufferedAmount
на месте. Нода мультипользовательская, с веб-панелью администратора и профилями
по ссылке `rookery://…` (как у happ/v2ray). Клиент есть в двух видах: headless CLI
и Wails GUI (окно, трей, автозапуск, импорт профиля по ссылке).

## Структура репозитория

- `shared/` — код, общий для ноды и клиента: формат заголовка адресата (`protocol`),
  HMAC-аутентификация сигналинга (`signaling`), кодек ссылки `rookery://` (`profile`)
- `node/` — серверная нода (`cmd/rookeryd`): сигналинг, релей, SQLite-хранилище
  пользователей (`internal/store`), веб-панель администратора (`internal/admin`).
  Собирается `CGO_ENABLED=0` под linux/amd64, публикуется как Docker-образ
- `client/` — Windows-клиент: чистое ядро (`internal/engine`), headless CLI
  (`cmd/rookery-cli`) и Wails GUI (`gui/`)

Модули объединены через `go.work` в корне; `shared` не импортирует `node`/`client`,
а `node` и `client` не импортируют друг друга.

## Как это устроено

Нода хранит пользователей в SQLite (`data_dir/rookery.db`, по умолчанию `/data`):
у каждого — свой `id` и HMAC-секрет. Панель администратора (`/admin`) создаёт
пользователей и выдаёт для каждого ссылку `rookery://…` — она содержит адрес ноды,
`id` и секрет в одном base64-блоке. В клиенте достаточно вставить эту ссылку
в настройках — вводить IP/порт/секрет вручную не нужно.

## Сборка

Требуется Go 1.22+. Артефакты собираются независимо:

```
make build-node    # bin/rookeryd — статический линукс-бинарь, CGO_ENABLED=0
make build-cli     # bin/rookery-cli — headless клиент
make build-gui     # GUI на Wails; см. раздел ниже про требования
make docker-build  # локальная сборка Docker-образа ноды (для теста, без публикации)
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

## Установка ноды на Ubuntu (Docker, рекомендуется)

При каждом пуше в `master`, затрагивающем `node/` или `shared/`, GitHub Actions
собирает образ ноды и публикует его в `ghcr.io/proxt/rookery-node` (см.
`.github/workflows/node-docker.yml`). На VDS ничего собирать не нужно, конфиг
редактировать тоже — нода сама генерирует логин администратора и per-user секреты
при первом запуске.

1. Поставить Docker, если его ещё нет:
   ```
   curl -fsSL https://get.docker.com | sudo sh
   ```
2. Скачать compose-файл:
   ```
   mkdir -p ~/rookery && cd ~/rookery
   curl -O https://raw.githubusercontent.com/proxt/rookery/master/node/deploy/docker-compose.yml
   ```
3. Запустить:
   ```
   sudo docker compose up -d
   sudo docker compose logs
   ```
   В логах при первом запуске будет строка вида
   `generated admin panel credentials ... username=admin password=...` —
   сохраните пароль, второй раз он не покажется (сменить его можно потом в самой панели).
4. Поставить Caddy на хосте (не в контейнере) и взять за основу
   `node/deploy/Caddyfile.example` — подставить свой домен вместо
   `your-node-hostname.example.com`. Caddy получит TLS-сертификат сам и
   проксирует на `127.0.0.1:8080`, куда `docker-compose.yml` прокидывает ноду
   через `network_mode: host`. Без Caddy панель и сигналинг снаружи не видны —
   для быстрой проверки можно временно зайти по SSH-туннелю на `127.0.0.1:8080/admin`.
5. Зайти на `https://ваш-домен/admin`, авторизоваться, при необходимости
   поправить «Публичный адрес ноды» в настройках (он же автоматически
   заполняется определённым IP сервера при первом старте — замените на домен).
   Создать пользователя, скопировать его ссылку `rookery://…`.
6. В клиенте (GUI → Настройки, или CLI-конфиг) вставить эту ссылку — адрес
   ноды, `id` и секрет заполнятся сами.

Обновление до новой версии: `sudo docker compose pull && sudo docker compose up -d`.
Данные (пользователи, логин администратора) переживают обновление — они лежат
в volume `rookery_data`, а не в образе.

`network_mode: host` в compose-файле не опционален — без него ICE-агент
объявит клиентам внутренний Docker-адрес контейнера вместо публичного IP VDS,
и подключение не установится.

Пакет `ghcr.io/proxt/rookery-node` публичный, `docker pull` на VDS работает
без `docker login`. Если у себя в форке пакет окажется приватным — сделать
публичным можно на странице пакета в GitHub (Packages → rookery-node →
Package settings → Change visibility).

### Локальная сборка из исходников (для разработки/тестов)

Без Docker, например чтобы погонять ноду в WSL при разработке:

```
git clone https://github.com/proxt/rookery.git
cd rookery
make build-node   # bin/rookeryd — статический линукс-бинарь, CGO_ENABLED=0
```

Дальше — как в разделе «Развёртывание ноды без Docker» ниже.

## Развёртывание ноды без Docker

1. Собрать (`make build-node`, см. выше) или скачать `bin/rookeryd`.
2. Скопировать в `/opt/rookery/rookeryd`:
   ```
   sudo mkdir -p /opt/rookery
   sudo cp bin/rookeryd /opt/rookery/rookeryd
   ```
3. Создать системного пользователя без прав на вход и каталог для данных:
   ```
   useradd --system --no-create-home --shell /usr/sbin/nologin rookery
   sudo mkdir -p /var/lib/rookery && sudo chown rookery:rookery /var/lib/rookery
   ```
4. По умолчанию нода пишет SQLite-базу в `/data` — для этого варианта переопределите
   `data_dir` в `/etc/rookery/node.yaml` (см. `node/configs/node.example.yaml`,
   все поля там опциональны):
   ```yaml
   data_dir: "/var/lib/rookery"
   ```
5. Поставить юнит `node/deploy/systemd/rookery-node.service` в `/etc/systemd/system/`
   и включить: `systemctl daemon-reload && systemctl enable --now rookery-node`.
   Юнит уже настроен с `NoNewPrivileges`, `ProtectSystem=strict`, `PrivateTmp`,
   `ReadWritePaths=/var/lib/rookery`, без лишних capabilities, и `Restart=always`.
6. Пароль администратора появится в `journalctl -u rookery-node` при первом запуске.
7. Поставить Caddy и взять за основу `node/deploy/Caddyfile.example` — подставить
   свой домен вместо `your-node-hostname.example.com`. Caddy сам получит TLS-сертификат
   и проксирует на `127.0.0.1:8080`, где слушает rookeryd.

### Порты

- **TCP 443** (или 80 для ACME HTTP-01 challenge) — снаружи, на Caddy.
- **UDP `ice_udp_port`** из конфига ноды (по умолчанию 51000) — снаружи, для WebRTC/ICE.
  Это единственный UDP-порт, который нужно открывать: он зафиксирован через
  `SetICEUDPMux`, диапазон эфемерных портов не используется.
- `listen_addr` ноды (по умолчанию `127.0.0.1:8080`) наружу открывать не нужно —
  это внутренний адрес, на который проксирует Caddy. Через него же доступна
  и панель администратора (`/admin`).

## Клиент: как подключиться

- **GUI**: Настройки → вставить ссылку `rookery://…` в поле «Ссылка профиля» →
  «Вставить» → «Сохранить». Дальше — большая кнопка «Подключить».
- **CLI**: скопировать `client/configs/client.example.yaml` в `client.yaml` и
  вписать `node_addr`, `user_id`, `secret` — их можно взять из панели
  администратора (или декодировать ссылку: это base64url JSON вида
  `{"n":"...","i":"...","s":"..."}` после `rookery://`).

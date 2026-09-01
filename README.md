# Rookery

Репозиторий: https://github.com/proxt/rookery

WebRTC-туннель: произвольный TCP/UDP поверх WebRTC DataChannel между Windows-клиентом
и нодой на Ubuntu, так что на уровне сети сессия выглядит как обычный p2p-трафик.

Статус: сквозной туннель работает — SOCKS5 CONNECT (TCP) и UDP ASSOCIATE проверены
end-to-end, реконнект с экспоненциальным backoff и backpressure по bufferedAmount
на месте. Клиент есть в двух видах: headless CLI и Wails GUI (окно, трей, автозапуск).

## Структура репозитория

- `shared/` — код, общий для ноды и клиента: формат заголовка адресата (`protocol`) и
  HMAC-аутентификация сигналинга (`signaling`)
- `node/` — серверная нода (`cmd/rookeryd`), собирается `CGO_ENABLED=0` под linux/amd64
- `client/` — Windows-клиент: чистое ядро (`internal/engine`), headless CLI
  (`cmd/rookery-cli`) и Wails GUI (`gui/`)

Модули объединены через `go.work` в корне; `shared` не импортирует `node`/`client`,
а `node` и `client` не импортируют друг друга.

## Сборка

Требуется Go 1.22+. Три артефакта собираются независимо:

```
make build-node   # bin/rookeryd — статический линукс-бинарь, CGO_ENABLED=0
make build-cli    # bin/rookery-cli — headless клиент
make build-gui    # GUI на Wails; см. раздел ниже про требования
```

```
make test   # go test по всем трём модулям
make lint   # go vet по всем трём модулям
make clean  # удалить bin/
```

### GUI (Wails)

`build-gui` требует Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
и **WebView2 Runtime**. Собирать нужно на Windows или в CI на `windows-latest` — кросс-компиляция
GUI с CGO-зависимостями (`fyne.io/systray`, `wailsapp/go-webview2`) с Linux не поддерживается.

## Конфигурация

Примеры конфигов лежат в `node/configs/node.example.yaml` и `client/configs/client.example.yaml`.
Скопируйте их без суффикса `.example` — реальные конфиги в `.gitignore` и не попадают в репозиторий.

Секрет для HMAC-подписи сигналинга можно задать прямо в конфиге (`secret`) или через
переменную окружения — укажите её имя в `secret_env`, и значение будет прочитано из неё
(это позволяет не хранить секрет на диске). Сгенерировать случайный секрет:

```
openssl rand -hex 32
```

Один и тот же секрет должен быть прописан и в конфиге ноды, и в конфиге клиента.

## Установка ноды на Ubuntu

Ставится прямо на сервере, сборка из исходников — бинарь не публикуется отдельно.

1. Поставить Go 1.22+, если его ещё нет (пакет в apt часто устаревший — надёжнее
   официальный тарбол; замените версию на актуальную с https://go.dev/dl/):
   ```
   curl -LO https://go.dev/dl/go1.27.0.linux-amd64.tar.gz
   sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.27.0.linux-amd64.tar.gz
   echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc && source ~/.bashrc
   go version
   ```
2. Склонировать репозиторий и собрать ноду:
   ```
   git clone https://github.com/proxt/rookery.git
   cd rookery
   make build-node
   ```
   Получится статический линукс-бинарь `bin/rookeryd` (`CGO_ENABLED=0`, без внешних
   зависимостей — файл можно скопировать на другую машину и без Go).
3. Дальше — как в разделе «Развёртывание ноды» ниже: системный пользователь,
   конфиг, systemd, Caddy.

Обновление до новой версии: `git pull && make build-node`, затем
`sudo systemctl restart rookery-node`.

## Развёртывание ноды

1. Бинарь уже собран на предыдущем шаге (`bin/rookeryd`, статический, без CGO).
2. Скопировать в `/opt/rookery/rookeryd`:
   ```
   sudo mkdir -p /opt/rookery
   sudo cp bin/rookeryd /opt/rookery/rookeryd
   ```
3. Создать системного пользователя без прав на вход:
   ```
   useradd --system --no-create-home --shell /usr/sbin/nologin rookery
   ```
4. Положить реальный конфиг в `/etc/rookery/node.yaml` (на основе
   `node/configs/node.example.yaml`), сгенерировать секрет (`openssl rand -hex 32`)
   и вписать его в `secret` конфигов ноды и клиента.
5. Поставить юнит `node/deploy/systemd/rookery-node.service` в `/etc/systemd/system/`
   и включить: `systemctl daemon-reload && systemctl enable --now rookery-node`.
   Юнит уже настроен с `NoNewPrivileges`, `ProtectSystem=strict`, `PrivateTmp`,
   без лишних capabilities, и `Restart=always`.
6. Поставить Caddy и взять за основу `node/deploy/Caddyfile.example` — подставить
   свой домен вместо `your-node-hostname.example.com`. Caddy сам получит TLS-сертификат
   и проксирует на `127.0.0.1:8080`, где слушает rookeryd.

### Порты

- **TCP 443** (или 80 для ACME HTTP-01 challenge) — снаружи, на Caddy.
- **UDP `ice_udp_port`** из конфига ноды (например 51000) — снаружи, для WebRTC/ICE.
  Это единственный UDP-порт, который нужно открывать: он зафиксирован через
  `SetICEUDPMux`, диапазон эфемерных портов не используется.
- `listen_addr` ноды (по умолчанию `127.0.0.1:8080`) наружу открывать не нужно —
  это внутренний адрес, на который проксирует Caddy.

# MusicPort

Desktop-приложение на Go и Fyne для Spotify-коллекции.

## Запуск

Скопируйте config.example.yaml в config.yaml, укажите spotify.client_id и зарегистрируйте Redirect URI http://127.0.0.1:8888/callback в Spotify Developer Dashboard.

```powershell
Copy-Item config.example.yaml config.yaml
./scripts/run.ps1
```

Окно содержит кнопки входа в Spotify, обновления плейлистов и загрузки треков, выбор коллекции, поиск и отмену запроса. По умолчанию выбрана коллекция «Мне нравится». Вход открывается в системном браузере, токены остаются в keyring. Если вход уже выполнен, нажмите «Загрузить треки».

Источник аудио пока не подключён: GUI показывает реальные метаданные Spotify, но не скачивает полные аудиофайлы.

## Сборка Windows

Fyne требует CGO и GCC (MinGW-w64). Скрипт использует portable GCC из .tools/mingw64/bin, если он есть; иначе gcc должен быть в PATH. Скрипт меняет переменные только на время своего выполнения.

```powershell
./scripts/build.ps1
```

Либо при установленном GCC:

```powershell
$env:CGO_ENABLED = '1'
go run .
go build -ldflags '-H=windowsgui' -o musicport.exe .
```

## Конфигурация и CLI

-config указывает YAML-файл. Приоритет: явно заданные флаги → SPOTIFY_CLIENT_ID → YAML → defaults. Старый YAML-параметр tui принимается для совместимости и игнорируется. GUI запускается по умолчанию. CLI доступен явно:

```powershell
go run . -login
go run . -list
go run . -headless -collections liked -dry-run
```

## Тесты

GUI тестируется программным драйвером Fyne без рабочего стола. Моки Service генерируются minimock.

```powershell
$env:GOFLAGS = '-tags=ci'
go generate ./...
go test -tags 'ci,integration' ./...
go vet ./...
go build .
```

Тег ci отключает desktop-драйвер, поэтому такой бинарник проверяет сборку, но не открывает окно. Для desktop используйте scripts/build.ps1 без этого тега. Unit-тесты не обращаются к реальному Spotify и keyring. Реальный OAuth проверяется вручную.

# MusicPort

Go-приложение для авторизации Spotify и импорта коллекции «Мне нравится» и доступных плейлистов.

## Запуск

Скопируйте `config.example.yaml` в `config.yaml` и заполните `spotify.client_id`. Настройки загружает отдельный пакет `internal/config`.

```powershell
Copy-Item config.example.yaml config.yaml
go run . -login
go run .
```

В примере включён TUI и выбрана коллекция `liked`. Другой файл можно указать через `-config settings.yaml`. Приоритет: явно заданные флаги → `SPOTIFY_CLIENT_ID` → YAML → значения по умолчанию. Для отключения TUI используйте `-tui=false`.

Без стандартного config.yaml запуск с флагами допустим; явно указанный файл должен существовать. Неизвестные поля и неверные настройки вызывают ошибку. Путь library отсчитывается от рабочей папки. Токены сохраняются только в keyring. При наличии config.yaml запуск без аргументов использует его настройки.

Создайте приложение в Spotify Developer Dashboard, зарегистрируйте Redirect URI `http://127.0.0.1:8888/callback` и укажите Client ID:

```powershell
$env:SPOTIFY_CLIENT_ID = 'your-client-id'
go run . -login
go run . -collections liked -tui
```

Откройте ссылку из команды login в браузере и разрешите доступ. Токены сохраняются в системном keyring. В TUI Enter запускает обработку коллекции, q отменяет работу или закрывает интерфейс после завершения.

```powershell
go run . -list
go run . -collections liked,PLAYLIST_ID -dry-run
```

По умолчанию используется «Мне нравится». Запуск без аргументов и без config.yaml показывает справку. Полные аудиофайлы Spotify API не предоставляет; адаптер внешнего аудиоисточника пока не подключён. Импортированные треки отображаются со статусом unresolved. Ручные файлы со ссылками не требуются.

## Проверки

```powershell
go generate ./...
go test ./...
go test -tags=integration ./...
go vet ./...
go build .
```

Реализованы SQLite-индекс, проверка SHA-256, блокировка второго процесса и восстановление опубликованных файлов в ядре загрузчика. Тесты minimock и интеграционные тесты работают без обращения к Spotify. Реальный OAuth требует вашего приложения Spotify.


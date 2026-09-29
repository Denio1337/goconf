# goenv

[![Go Reference](https://pkg.go.dev/badge/github.com/Denio1337/goenv.svg)](https://pkg.go.dev/github.com/Denio1337/goenv)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/Denio1337/goenv)](https://goreportcard.com/report/github.com/Denio1337/goenv)

**goenv** — идиоматическая, строго типизированная библиотека на языке Go для загрузки и валидации конфигурации из различных источников в единую структуру (`struct`).

Проект спроектирован по модульной архитектуре с поддержкой расширяемых провайдеров (`Source`). Из коробки полностью реализован продвинутый парсер файлов `.env`, а добавление новых форматов (JSON, YAML, TOML, переменные окружения ОС, Consul, Vault и т.д.) выполняется реализацией одного компактного интерфейса.

---

## Ключевые возможности

- 🛡️ **Строгая схема и контроль типов**: Никаких скрытых ошибок преобразования типов во время работы приложения. Если поле ожидает `int`, а передано `"abc"`, библиотека выдаст подробную ошибку.
- 📋 **Агрегация ошибок (Multi-Error Reporting)**: Библиотека не прерывает работу на первой ошибке, а собирает все невалидные поля конфигурации в единый структурированный отчёт, где указан путь в структуре, ключ источника, переданное значение и причина ошибки.
- 🔌 **Расширяемая архитектура источников (`Source`)**: Возможность комбинировать и переопределять конфигурации из нескольких источников с разным приоритетом.
- 📝 **Полноценный парсер `.env`**:
  - Одинарные (`'...'`) и двойные (`"..."`) кавычки
  - Многострочные значения (сертификаты, RSA-ключи и т.п.)
  - Экранирование (`\n`, `\t`, `\"`, `\\`)
  - Комментарии (`#`) как на отдельных строках, так и inline
  - Поддержка префикса `export `
  - Интерполяция переменных (`${HOST}:${PORT}`, `$VAR`, `${VAR:-default}`)
- 🧩 **Богатая поддержка типов Go**:
  - Примитивные типы (`int*`, `uint*`, `float*`, `bool`, `string`)
  - Временные интервалы (`time.Duration`, например `15s`, `5m`)
  - Даты и время (`time.Time` с авто-определением форматов или тегом `layout`)
  - Сетевые типы (`net.IP`, `*url.URL`)
  - Срезы (`[]string`, `[]int`, `[]time.Duration` с настраиваемым разделителем `sep`)
  - Словари (`map[string]T`)
  - Указатели на любые поддерживаемые типы (с выделением памяти только при наличии значения)
  - Вложенные и анонимные структуры с поддержкой префиксов (`env-prefix`)
  - Пользовательские типы через интерфейс `encoding.TextUnmarshaler`
  - Кастомная валидация через интерфейс `Validator`
- ⚡ **Zero Dependencies**: Реализовано исключительно на стандартной библиотеке Go.

---

## Установка

```bash
go get github.com/Denio1337/goenv
```

Требуется версия Go 1.20 или новее.

---

## Быстрый старт

Создайте файл `.env`:

```env
APP_NAME=MyService
ENVIRONMENT=production
DEBUG=false

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_TIMEOUT=30s

DATABASE_HOST=postgres.internal
DATABASE_PORT=5432
DATABASE_PASSWORD=secret-password
```

Опишите конфигурационную структуру и загрузите её:

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/Denio1337/goenv"
)

type ServerConfig struct {
	Host    string        `env:"HOST" default:"localhost"`
	Port    int           `env:"PORT" default:"8080"`
	Timeout time.Duration `env:"TIMEOUT" default:"10s"`
}

type DatabaseConfig struct {
	Host     string `env:"HOST"`
	Port     int    `env:"PORT" default:"5432"`
	Password string `env:"PASSWORD" required:"true"`
}

type Config struct {
	AppName     string         `env:"APP_NAME"`
	Environment string         `env:"ENVIRONMENT" default:"development"`
	Debug       bool           `env:"DEBUG"`
	Server      ServerConfig   `env-prefix:"SERVER_"`
	Database    DatabaseConfig `env-prefix:"DATABASE_"`
}

func main() {
	var cfg Config

	// Загрузка конфигурации из .env файла
	if err := goenv.Load(&cfg, goenv.WithDotEnv(".env")); err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	fmt.Printf("Сервис: %s (%s)\n", cfg.AppName, cfg.Environment)
	fmt.Printf("Сервер запущен на %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("БД хост: %s:%d\n", cfg.Database.Host, cfg.Database.Port)
}
```

---

## Теги структуры

| Тег | Описание | Пример |
|---|---|---|
| `env` / `config` | Имя ключа в источнике конфигурации | `env:"PORT"` |
| `default` / `env-default` | Значение по умолчанию, если ключ отсутствует или пуст | `default:"8080"` |
| `required` / `env-required` | Обязательное поле. Возвращает ошибку, если не задано | `required:"true"` |
| `env-prefix` / `prefix` | Префикс ключей для вложенной структуры | `env-prefix:"DB_"` |
| `sep` | Разделитель для срезов и словарей (по умолчанию `,`) | `sep:";"` |
| `layout` | Формат времени для парсинга `time.Time` | `layout:"2006-01-02"` |

Также поддерживается краткая запись опций через запятую в теге `env`:
```go
type ServerConfig struct {
    Port int `env:"PORT,required,default=8080"`
}
```

---

## Строгая схема и отчёт об ошибках

Если в конфигурации допущены ошибки типизации или пропущены обязательные поля, `goenv` возвращает ошибку `*ValidationError`, содержащую список всех некорректных полей.

### Пример некорректного `.env`:
```env
PORT=invalid_number
TIMEOUT=invalid_duration
DEBUG=not_a_bool
# DATABASE_PASSWORD отсутствует, хотя помечен required:"true"
```

### Форматированный вывод ошибки:
```text
goenv: schema validation failed with 4 error(s):
  [1] field "Server.Port" (key "SERVER_PORT") with value "invalid_number": cannot convert to int: expected integer, got "invalid_number": strconv.ParseInt: parsing "invalid_number": invalid syntax
  [2] field "Server.Timeout" (key "SERVER_TIMEOUT") with value "invalid_duration": cannot convert to time.Duration: invalid duration "invalid_duration": time: invalid duration "invalid_duration"
  [3] field "Debug" (key "DEBUG") with value "not_a_bool": cannot convert to bool: expected boolean (true/false/1/0), got "not_a_bool": strconv.ParseBool: parsing "not_a_bool": invalid syntax
  [4] field "Database.Password" (key "DATABASE_PASSWORD"): required field is missing or empty
```

### Программная инспекция ошибок:

```go
var valErr *goenv.ValidationError
if errors.As(err, &valErr) {
    for _, fe := range valErr.Errors {
        fmt.Println("Поле:", fe.Field)       // e.g. "Server.Port"
        fmt.Println("Ключ:", fe.Key)          // e.g. "SERVER_PORT"
        fmt.Println("Значение:", fe.Value)    // e.g. "invalid_number"
        fmt.Println("Ожидалось:", fe.TargetType) // e.g. "int"
        fmt.Println("Причина:", fe.Err)       // e.g. strconv.ErrSyntax
    }
}
```

Ошибки поддерживают проверку через стандартный механизм `errors.Is`:
```go
if errors.Is(err, goenv.ErrMissingRequired) {
    // обработка отсутствующих обязательных полей
}
```

---

## Архитектура расширения форматов (`Source`)

Любой формат или провайдер конфигурации (JSON, YAML, TOML, переменные окружения ОС, etcd, Consul) реализует интерфейс:

```go
type Source interface {
    Name() string
    Load(ctx context.Context) (map[string]any, error)
}
```

### Пример реализации JSON источника:

```go
type JSONSource struct {
    path string
}

func (s *JSONSource) Name() string { return "json:" + s.path }

func (s *JSONSource) Load(ctx context.Context) (map[string]any, error) {
    b, err := os.ReadFile(s.path)
    if err != nil {
        return nil, err
    }
    var data map[string]any
    if err := json.Unmarshal(b, &data); err != nil {
        return nil, err
    }
    return data, nil
}
```

### Использование нескольких источников с приоритетами:

```go
err := goenv.Load(&cfg,
    goenv.WithDotEnv(".env"),                  // Базовые значения из .env
    goenv.WithSource(NewJSONSource("cfg.json")),// Переопределения из JSON
)
```
Источники применяются по порядку: более поздние перезаписывают совпавшие ключи более ранних.

---

## Пользовательская валидация (`Validator`)

Любая структура может реализовать интерфейс `Validator` для дополнительной бизнес-проверки после разбора:

```go
type ServerConfig struct {
    Port int `env:"PORT" default:"8080"`
}

func (s *ServerConfig) Validate() error {
    if s.Port < 1024 || s.Port > 65535 {
        return fmt.Errorf("port %d is out of allowed user range (1024-65535)", s.Port)
    }
    return nil
}
```

---

## Структура репозитория

```
goenv/
├── go.mod
├── goenv.go               # Основной фасад: Load, MustLoad, Loader
├── options.go             # Функциональные опции (WithDotEnv, WithSource, etc.)
├── errors.go              # ValidationError, FieldError, sentinel errors
├── decoder.go             # Строгий рефлексивный декодер схемы
├── source.go              # Интерфейс Source и MapSource
├── store/                 # Пакет хранилища конфигурации (store.New)
│   ├── store.go           # Потокобезопасное хранилище и поиск ключей
│   └── store_test.go      # Тесты хранилища
├── source/
│   ├── dotenv/            # Провайдер формата .env (dotenv.New)
│   │   ├── parser.go      # Лексер и парсер .env с интерполяцией
│   │   └── dotenv.go      # Реализация Source для .env
│   └── mapsource/         # Провайдер in-memory map (mapsource.New)
│       ├── mapsource.go   # Реализация Source для map
│       └── mapsource_test.go
├── examples/              # Готовые примеры использования
│   ├── basic/             # Базовый пример с .env
│   ├── validation_errors/ # Демонстрация строгой валидации
│   └── custom_source/     # Демонстрация добавления JSON источника
├── decoder_test.go        # Тесты декодирования и типов
├── goenv_test.go          # Интеграционные тесты
└── LICENSE                # Лицензия MIT
```

---

## Лицензия

Распространяется под лицензией MIT. Подробности в файле [LICENSE](LICENSE).

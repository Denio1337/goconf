# goconf

[![Go Reference](https://pkg.go.dev/badge/github.com/Denio1337/goconf.svg)](https://pkg.go.dev/github.com/Denio1337/goconf)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/Denio1337/goconf)](https://goreportcard.com/report/github.com/Denio1337/goconf)

**goconf** — библиотека на языке Go для загрузки и валидации конфигурации из различных источников в единую структуру (`struct`).

Проект спроектирован по модульной архитектуре с поддержкой расширяемых провайдеров (`Source`). Из коробки полностью реализован продвинутый парсер файлов `.env`, а добавление новых форматов (JSON, YAML, TOML, переменные окружения ОС, Consul, Vault и т.д.) выполняется реализацией одного компактного интерфейса.

---

## Ключевые возможности

- 🛡️ **Строгая схема и контроль типов**: Никаких скрытых ошибок преобразования типов во время работы приложения. Если поле ожидает `int`, а передано `"abc"`, библиотека выдаст подробную ошибку.
- 📋 **Агрегация ошибок (Multi-Error Reporting)**: Библиотека не прерывает работу на первой ошибке, а собирает все невалидные поля конфигурации в единый структурированный отчёт, где указан путь в структуре, ключ источника, переданное значение и причина ошибки.
- 🔌 **Расширяемая архитектура источников (`Source`)**: Возможность комбинировать и переопределять конфигурации из нескольких источников с разным приоритетом.
- 📁 **Встроенные провайдеры форматов**:
  - **`.env`**: кавычки, многострочные значения, экранирование, inline-комментарии, интерполяция `${VAR:-default}`.
  - **`.ini`**: секции `[section]`, подсекции `[section.sub]`, комментарии `;` и `#`, многоуровневый мапинг.
  - **`.json`**: иерархические JSON-документы, сохранение точности чисел (`json.Number`), вложенные объекты.
  - **`mapsource`**: in-memory структуры для программных настроек и тестов.
- 🧩 **Богатая поддержка типов Go**:
  - Примитивные типы (`int*`, `uint*`, `float*`, `bool`, `string`)
  - Временные интервалы (`time.Duration`, например `15s`, `5m`)
  - Даты и время (`time.Time` с авто-определением форматов или тегом `layout`)
  - Сетевые типы (`net.IP`, `*url.URL`)
  - Срезы (`[]string`, `[]int`, `[]time.Duration` с настраиваемым разделителем `sep`)
  - Словари (`map[string]T`)
  - Указатели на любые поддерживаемые типы (с выделением памяти только при наличии значения)
  - Вложенные и анонимные структуры с поддержкой префиксов (`prefix`)
  - Пользовательские типы через интерфейс `encoding.TextUnmarshaler`
  - Кастомная валидация через интерфейс `Validator`
- ⚡ **Zero Dependencies**: Реализовано исключительно на стандартной библиотеке Go.

---

## Установка

```bash
go get github.com/Denio1337/goconf
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

	"github.com/Denio1337/goconf"
)

type ServerConfig struct {
	Host    string        `key:"HOST" default:"localhost"`
	Port    int           `key:"PORT" default:"8080"`
	Timeout time.Duration `key:"TIMEOUT" default:"10s"`
}

type DatabaseConfig struct {
	Host     string `key:"HOST"`
	Port     int    `key:"PORT" default:"5432"`
	Password string `key:"PASSWORD" required:"true"`
}

type Config struct {
	AppName     string         `key:"APP_NAME"`
	Environment string         `key:"ENVIRONMENT" default:"development"`
	Debug       bool           `key:"DEBUG"`
	Server      ServerConfig   `prefix:"SERVER_"`
	Database    DatabaseConfig `prefix:"DATABASE_"`
}

func main() {
	var cfg Config

	// Загрузка конфигурации из .env файла
	if err := goconf.Load(&cfg, goconf.WithDotEnv(".env")); err != nil {
		log.Fatalf("Ошибка конфигурации: %v", err)
	}

	fmt.Printf("Сервис: %s (%s)\n", cfg.AppName, cfg.Environment)
	fmt.Printf("Сервер запущен на %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("БД хост: %s:%d\n", cfg.Database.Host, cfg.Database.Port)
}
```

---

## Теги структуры и константы

Все теги зафиксированы в коде в виде экспортируемых констант (файл `tags.go`):

| Константа | Имя тега | Описание | Пример |
|---|---|---|---|
| `goconf.TagKey` | `key` | Имя ключа в источнике конфигурации | `key:"PORT"` |
| `goconf.TagDefault` | `default` | Значение по умолчанию, если ключ отсутствует или пуст | `default:"8080"` |
| `goconf.TagRequired` | `required` | Обязательное поле. Возвращает ошибку, если не задано | `required:"true"` |
| `goconf.TagPrefix` | `prefix` | Префикс ключей для вложенной структуры | `prefix:"DB_"` |
| `goconf.TagSep` | `sep` | Разделитель для срезов и словарей (по умолчанию `,`) | `sep:";"` |
| `goconf.TagLayout` | `layout` | Формат времени для парсинга `time.Time` | `layout:"2006-01-02"` |

### Особенности работы с `prefix`:
1. **Явный префикс**: `prefix:"DB_"` добавляет `DB_` ко всем дочерним полям вложенной структуры (`HOST` -> `DB_HOST`).
2. **Использование `key` как префикса**: если на вложенную структуру повесить `key:"DATABASE"`, она автоматически сформирует префикс `DATABASE_`.
3. **Отключение автопрефикса**: `prefix:""` явно отключает автопрефикс у именованной структуры, делая её поля плоскими.
4. **Абсолютные ключи**: если дочернее поле начинается со слэша `/` (например, `key:"/GLOBAL_SECRET"`), оно **игнорирует любые префиксы** родителей.
5. **Глобальный префикс**: опция `goconf.WithPrefix("APP_")` добавляет префикс ко всем ключам корневой структуры.

### Краткая запись опций в теге `key`:
```go
type ServerConfig struct {
    Port int `key:"PORT,required,default=8080"`
}
```

### Программная инспекция ошибок:

```go
var valErr *goconf.ValidationError
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
if errors.Is(err, goconf.ErrMissingRequired) {
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

### Использование файлов INI:

```ini
# config.ini
app_name = "My Application"

[server]
host = 0.0.0.0
port = 8080
timeout = 30s

[database]
host = localhost
port = 5432
password = "secret"
```

### Использование файлов JSON:

```json
{
  "app_name": "My Application",
  "server": {
    "host": "0.0.0.0",
    "port": 8080
  },
  "database": {
    "host": "localhost",
    "port": 5432,
    "password": "secret"
  }
}
```

```go
err := goconf.Load(&cfg, goconf.WithJSON("config.json"))
```

### Использование файлов YAML:

```yaml
app_name: "My Application"
server:
  host: 0.0.0.0
  port: 8080
database:
  host: localhost
  port: 5432
  password: "secret"
```

```go
err := goconf.Load(&cfg, goconf.WithYAML("config.yaml"))
```

### Использование нескольких источников с приоритетами:

```go
err := goconf.Load(&cfg,
    goconf.WithDotEnv(".env"),     // 1. Базовые значения из .env
    goconf.WithINI("config.ini"),  // 2. Переопределения из INI
    goconf.WithJSON("local.json"), // 3. Локальные переопределения из JSON
    goconf.WithYAML("prod.yaml"),  // 4. Финальные переопределения из YAML
)
```
Источники применяются по порядку: более поздние перезаписывают совпавшие ключи более ранних.

---

## Пользовательская валидация (`Validator`)

Любая структура может реализовать интерфейс `Validator` для дополнительной бизнес-проверки после разбора:

```go
type ServerConfig struct {
    Port int `key:"PORT" default:"8080"`
}

func (s *ServerConfig) Validate() error {
    if s.Port < 1024 || s.Port > 65535 {
        return fmt.Errorf("port %d is out of allowed user range (1024-65535)", s.Port)
    }
    return nil
}
```

## Лицензия

Распространяется под лицензией MIT. Подробности в файле [LICENSE](LICENSE).

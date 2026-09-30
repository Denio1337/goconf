# goconf

[![Go Reference](https://pkg.go.dev/badge/github.com/Denio1337/goconf.svg)](https://pkg.go.dev/github.com/Denio1337/goconf)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[English](README.md) | [Русский](README.ru.md)

**goconf** — современная, надёжная и строго типизированная библиотека для работы с конфигурацией на языке Go. Она загружает и объединяет конфигурации из различных источников (`.env`, `INI`, `JSON`, `YAML`, `TOML`, переменные окружения ОС или пользовательские провайдеры) в целевую структуру Go с полной проверкой типов и детерминированным каскадным приоритетом.

## Быстрый старт

### Установка

```bash
go get github.com/Denio1337/goconf
```

> **Требования**: Go **1.22.0** или выше.

### Использование

Создайте файл `.env` (или задайте переменные окружения в вашей системе/контейнере):

```env
APP_NAME=MyService
SERVER_PORT=8080
SERVER_READ_TIMEOUT=10s
DATABASE_HOST=postgres.internal
DATABASE_PASSWORD=super-secret-production-password
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

// ServerConfig демонстрирует вложенные структуры и валидацию через Validator.
// Обратите внимание: теги `key` писать необязательно! Имена вроде SERVER_READ_TIMEOUT вычисляются автоматически.
type ServerConfig struct {
	Host         string        `default:"localhost"`
	Port         int           `default:"8080"`
	ReadTimeout  time.Duration `default:"5s"`
	WriteTimeout time.Duration `default:"10s"`
}

// Validate реализует интерфейс goconf.Validator для проверки бизнес-правил после загрузки.
func (s *ServerConfig) Validate() error {
	if s.Port < 1024 || s.Port > 65535 {
		return fmt.Errorf("server port %d must be in range 1024-65535", s.Port)
	}
	return nil
}

type DatabaseConfig struct {
	Host     string                `default:"localhost"`
	Port     int                   `default:"5432"`
	User     string                `default:"postgres"`
	Password goconf.Secret[string] `required:"true"` // Защита от случайных утечек в логи
}

type Config struct {
	AppName  string `default:"MyService"`
	Debug    bool   `default:"false"`
	Server   ServerConfig
	Database DatabaseConfig
}

func main() {
	var cfg Config

	// Загрузка конфигурации: проверяет переменные окружения ОС с наивысшим приоритетом,
	// затем файл .env (отсутствие .env автоматически игнорируется).
	if err := goconf.Load(&cfg); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	fmt.Println("--- Configuration Loaded Successfully ---")
	fmt.Printf("App: %s (Debug: %t)\n", cfg.AppName, cfg.Debug)
	fmt.Printf("Server: %s:%d (ReadTimeout: %v)\n", cfg.Server.Host, cfg.Server.Port, cfg.Server.ReadTimeout)
	fmt.Printf("Database: %s@%s:%d\n\n", cfg.Database.User, cfg.Database.Host, cfg.Database.Port)

	// Демонстрация безопасного вывода с goconf.Secret[T]:
	fmt.Println("--- Sensitive Data Protection (Secret[T]) ---")
	fmt.Printf("Database struct dump (%%+v) : %+v\n", cfg.Database)
	fmt.Printf("Direct field print (%%s)     : %s\n", cfg.Database.Password)
	fmt.Printf("Raw value via .Value()      : %s\n", cfg.Database.Password.Value())
}
```

#### Результат вывода:

```text
--- Configuration Loaded Successfully ---
App: MyService (Debug: false)
Server: localhost:8080 (ReadTimeout: 10s)
Database: postgres@postgres.internal:5432

--- Sensitive Data Protection (Secret[T]) ---
Database struct dump (%+v) : {Host:postgres.internal Port:5432 User:postgres Password:[SECRET]}
Direct field print (%s)     : [SECRET]
Raw value via .Value()      : super-secret-production-password
```

## Ключевые возможности

- 🛡️ **Строгая схема и контроль типов**: Никаких скрытых ошибок преобразования типов во время работы приложения. Каждое значение строго приводится к типу поля структуры.
- 📋 **Агрегация ошибок (Multi-Error Reporting)**: Библиотека не прерывает работу на первой ошибке, а собирает все ошибки валидации и несоответствия типов со всей структуры в единый структурированный отчёт.
- 🔄 **Каскадное переопределение (Cascading Multi-Source Overrides)**: Объединяйте несколько слоёв конфигурации с детерминированным приоритетом (например: базовый INI &rarr; общий YAML &rarr; сервисный TOML &rarr; локальный JSON &rarr; секреты `.env` &rarr; переменные окружения ОС).
- 🏷️ **Умное определение имён и канонический тег `key`**: Поддержка явных тегов `key`, а также автоматическое сопоставление полей без тегов (`snake_case`, `camelCase`, `kebab-case`, `UPPER_CASE`, `Section.Key`).
- 📁 **Встроенные источники форматов**:
  - **Переменные окружения ОС**: Полная поддержка 12-Factor приложений (`WithEnv()`, `WithEnvPrefix("APP_")`), отсечение префиксов и иерархия через двойное подчёркивание `__`. [Пример](examples/basic/).
  - **`.env`**: Поддержка кавычек, многострочных значений, экранирования, inline-комментариев и интерполяции `${VAR:-default}`. [Пример](examples/basic/main.go).
  - **`.ini`**: Секции `[section]`, подсекции `[section.sub]`, комментарии (`;` и `#`), иерархический маппинг. [Пример](examples/ini/main.go).
  - **`.json`**: Иерархические JSON-документы с сохранением точности чисел. [Пример](examples/json/main.go).
  - **`.yaml`**: Маппинг структур YAML. [Пример](examples/yaml/main.go).
  - **`.toml`**: Таблицы и значения TOML. [Пример](examples/toml/main.go).
- 🧩 **Богатая поддержка типов Go**:
  - Примитивные типы (`int`, `uint`, `float`, `bool`, `string`)
  - Временные интервалы (`time.Duration`, например `10s`, `5m`)
  - Даты и время (`time.Time` с авто-определением форматов или тегом `layout`)
  - Сетевые типы (`net.IP`, `url.URL`)
  - Срезы (`[]string`, `[]int`, `[]time.Duration` с настраиваемым разделителем `sep`)
  - Словари (`map[string]T`)
  - Указатели (выделяются только при наличии значения)
  - Вложенные и анонимные структуры с наследованием префиксов
  - Пользовательские типы через интерфейс `encoding.TextUnmarshaler`
- 🔒 **Защита конфиденциальных данных (`Secret[T]`)**: Дженерик-обёртка для защиты паролей, токенов и API-ключей. Значения маскируются как `[SECRET]` в `fmt.Print*`, `log.Print*` и при JSON-сериализации, но доступны в коде через `.Value()`.
- ✅ **Пользовательская валидация (`Validator`)**: Реализация интерфейса `Validate() error` на структурах для проверки бизнес-правил сразу после загрузки.
- 🪶 **Минимальные зависимости**: Ядро библиотеки написано исключительно на стандартной библиотеке Go, с компактными опциональными модулями для YAML и TOML.

## Каскадное переопределение источников

`goconf` позволяет наслаивать несколько источников с чётким приоритетом. Источники применяются последовательно: более поздние перезаписывают совпавшие ключи более ранних.

```go
goconf.WithINI("config.ini")   // 1. Базовые значения
goconf.WithYAML("config.yaml") // 2. Общая конфигурация сервиса
goconf.WithTOML("config.toml") // 3. Переопределения окружения
goconf.WithJSON("local.json")  // 4. Локальные переопределения разработки
goconf.WithDotEnv(".env")      // 5. Секреты разработчика
// Системные переменные окружения ОС всегда имеют наивысший приоритет
```

Полный рабочий пример каскада из 5 форматов доступен в [examples/complex](examples/complex).

## Теги структур и опции

Теги зафиксированы в виде констант в файле [tags.go](tags.go):

| Константа | Имя тега | Описание | Пример |
|---|---|---|---|
| `goconf.TagKey` | `key` | Имя ключа в источнике конфигурации | `key:"PORT"` |
| `goconf.TagDefault` | `default` | Значение по умолчанию, если ключ отсутствует или пуст | `default:"8080"` |
| `goconf.TagRequired` | `required` | Обязательное поле. Возвращает ошибку, если не задано | `required:"true"` |
| `goconf.TagPrefix` | `prefix` | Префикс ключей для вложенной структуры | `prefix:"DB_"` |
| `goconf.TagSep` | `sep` | Разделитель для срезов и словарей (по умолчанию `,`) | `sep:";"` |
| `goconf.TagLayout` | `layout` | Формат времени для парсинга `time.Time` | `layout:"2006-01-02"` |

### Правила работы с `prefix`:

1. **Явный префикс**: `prefix:"DB_"` добавляет `DB_` ко всем дочерним полям структуры (`HOST` &rarr; `DB_HOST`).
2. **`key` как префикс**: если на вложенную структуру повесить `key:"DATABASE"`, она автоматически сформирует префикс `DATABASE_`.
3. **Отключение автопрефикса**: `prefix:""` явно отключает префикс у структуры, делая её поля плоскими.
4. **Абсолютные ключи**: если дочернее поле начинается со слэша `/` (например, `key:"/GLOBAL_SECRET"`), оно **игнорирует любые префиксы** родителей.
5. **Глобальный префикс**: опция `goconf.WithPrefix("APP_")` добавляет префикс ко всем ключам корневой структуры.

## Инспекция ошибок (Multi-Error Reporting)

Вместо остановки на первой ошибке, `goconf` собирает все ошибки схемы и несовпадения типов в единый отчёт. См. [пример](examples/validation_errors/main.go).

## Защита конфиденциальных данных (`Secret[T]`)

Оборачивайте чувствительные поля (пароли, токены, приватные ключи) в `goconf.Secret[T]`, чтобы гарантировать защиту от случайной утечки в вывод терминала, логи приложения или JSON-ответы:

```go
type DatabaseConfig struct {
    Host     string                `key:"HOST"`
    Password goconf.Secret[string] `key:"PASSWORD"`
    Port     int                   `key:"PORT" default:"5432"`
}

func main() {
    var cfg DatabaseConfig
    goconf.Load(&cfg)

    // При выводе структуры или поля напрямую значение всегда маскируется как [SECRET]:
    fmt.Printf("%+v\n", cfg)      // Вывод: {Host:localhost Password:[SECRET] Port:5432}
    fmt.Println(cfg.Password)     // Вывод: [SECRET]
    log.Println(cfg)              // Вывод: {localhost [SECRET] 5432}

    // Безопасное получение реального значения при необходимости:
    rawPassword := cfg.Password.Value()
    db.Connect(rawPassword)
}
```

## Пользовательская валидация (`Validator`)

Любая структура может реализовать интерфейс `Validator` для дополнительной бизнес-проверки после парсинга:

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

## Расширяемая архитектура источников

Пользовательские провайдеры (etcd, HashiCorp Vault, AWS Secrets Manager, Kubernetes ConfigMaps) реализуют интерфейс `Source`:

```go
type Source interface {
    Name() string
    Load(ctx context.Context) (map[string]any, error)
}
```

### Пример реализации:

```go
type CustomSource struct {
    endpoint string
}

func (s *CustomSource) Name() string {
    return "custom:" + s.endpoint
}

func (s *CustomSource) Load(ctx context.Context) (map[string]any, error) {
    // Получение данных и возврат карты
    return map[string]any{
        "APP_NAME": "ClusterApp",
        "DATABASE_PORT": 5432,
    }, nil
}
```

Передайте источник напрямую в `goconf.Load`:

```go
err := goconf.Load(&cfg, goconf.WithSource(&CustomSource{endpoint: "https://api.internal"}))
```

## Лицензия

Распространяется под лицензией MIT. Подробности в файле [LICENSE](LICENSE).

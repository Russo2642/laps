# LAPS - Legal and Psychological Services

LAPS - это сервис для поиска квалифицированных юристов и психологов и записи на консультации к ним.

## Особенности

- Поиск юристов и психологов по специализациям
- Система записи на консультации
- Управление профилями специалистов
- Система отзывов и рейтингов
- JWT-аутентификация и авторизация

## Технический стек

- **Язык программирования**: Go 1.21+
- **Фреймворк**: Gin
- **База данных**: PostgreSQL
- **Логирование**: Zap
- **Аутентификация**: JWT
- **Зависимости**: управляются с помощью Go Modules

## Структура проекта

```
laps/
├── cmd/                     # Исполняемые файлы
│   └── api/                 # API сервер
├── config/                  # Конфигурации
├── internal/                # Внутренний код приложения
│   ├── domain/              # Модели и бизнес-объекты
│   ├── repository/          # Слой доступа к данным
│   ├── service/             # Бизнес-логика
│   └── transport/           # Транспортный слой
│       └── rest/            # REST API обработчики
├── migrations/              # SQL миграции для базы данных
├── pkg/                     # Общие пакеты
│   ├── database/            # Утилиты для работы с базой данных
│   ├── logger/              # Утилиты для логирования
│   └── validator/           # Валидаторы
├── .env                     # Конфигурационные переменные окружения
├── .gitignore               # Игнорируемые Git файлы
├── go.mod                   # Go модули
├── go.sum                   # Go модули checksum
├── main.go                  # Точка входа в приложение
└── README.md                # Документация проекта
```

## Запуск приложения

### Предварительные требования

- Go 1.21+
- PostgreSQL 14+

### Настройка окружения

1. Клонируйте репозиторий:
   ```bash
   git clone https://github.com/yourusername/laps.git
   cd laps
   ```

2. Создайте файл `.env` на основе `.env.example` и настройте переменные окружения:
   ```bash
   cp .env.example .env
   ```

3. Запустите PostgreSQL и создайте базу данных:
   ```sql
   CREATE DATABASE laps;
   ```

### Сборка и запуск

1. Установите зависимости:
   ```bash
   go mod download
   ```

2. Соберите приложение:
   ```bash
   go build -o laps
   ```

3. Запустите приложение:
   ```bash
   ./laps
   ```

## API Документация

API документация доступна по адресу `/swagger/index.html` при запущенном приложении в режиме разработки.

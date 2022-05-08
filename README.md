# Утилита "SQL Мигратор"

[Технические задание](SPECIFICATIONS.md) утилиты

## Установка утилиты

Текущая версия в разработке
```shell
go install github.com/bojik/sqlmigrator/cmd/gomigrator@develop
```

или

```shell
git clone -b develop git@github.com:bojik/sqlmigrator.git # клонируем ветку develop
cd sqlmigrator
make build # собираем утилиту 
./bin/gomigrator version # отображаем версию утилиты
```

## Тестирование
 
- `make test` - запуск unit тестов

## Интеграционное тестирование

1. `docker-compose up` - запустить контейнер с пустой БД
2. `make test-integration` - запустить интеграционные тесты, [конфигурация интеграционных тестов](pkg/migrator/testdata/config.yaml) 

## Генерация

- `make generate` - перегенерирует моки, текстовое представление констант, обновляет версию

## Использование утилиты

- `gomigrator init` - создаёт конфигурационный файл
- `gomigrator create` - генерирует файлы миграции
- `gomigrator up` - выполняет SQL миграции
- `gomigrator down` - откатывает последнюю SQL миграцию
- `gomigrator redo` - откатывает последнюю SQL миграцию и выполняет её заново
- `gomigrator status` - показывает текущий статус миграций в БД
- `gomigrator dbversion` - отображает последнюю версию миграции в БД


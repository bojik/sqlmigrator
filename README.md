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

### Пример использования go миграций

_**Важно!** Импортировать пакет с миграциями в приложение_

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bojik/sqlmigrator/pkg/migrator"
	// необходимо подключить пакет с миграциями
	_ "test-go-mig/m"
)

func main() {
	m := migrator.New(os.Stdout) // отправляем дебаг информацию в консоль
	rows, err := m.ApplyUpGoMigration(
		context.Background(),
		"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable",
	)
	if err != nil {
		panic(err.Error())
	}
	for _, row := range rows {
		fmt.Printf("%d %s %s", row.Version, row.Type.String(), row.SQL)
	}
}
```
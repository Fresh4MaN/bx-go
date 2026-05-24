# BitrixGo

Go-библиотека для прямой работы с MySQL-базой 1C-Bitrix: типизированный CRUD, фильтры в стиле Bitrix, связи между таблицами и upsert по внешнему идентификатору.

Работает **напрямую с MySQL**, без REST API Bitrix. Подходит для кастомных таблиц и Highload-блоков.

## Возможности

- Подключение к MySQL через DSN или `.env`
- Маппинг struct → таблица через теги `bx`
- CRUD и пакетные операции (`AddMulti`, `UpdateMulti`, `DeleteMulti`)
- Фильтры в стиле Bitrix (`=FIELD`, `!FIELD`, `>FIELD`, `%FIELD`, `@FIELD`)
- Preload связей отдельными SELECT (без JOIN)
- Many-to-one и one-to-many (inverse)
- Cascade: `AddCascade`, `SaveCascade`, `DeleteCascade`
- Upsert и поиск по внешнему коду (`ext`) — для интеграций без PK

## Требования

- Go 1.22+
- MySQL 5.7+ / 8.0 (база Bitrix с нужными таблицами)

## Установка

```bash
go get bitrixgo/bitrixgo
```

## Быстрый старт

### 1. Настройка подключения

Скопируйте `.env.example` в `.env` и укажите параметры вашей базы:

```env
BITRIX_DB_HOST=127.0.0.1
BITRIX_DB_PORT=3306
BITRIX_DB_USER=user
BITRIX_DB_PASSWORD=password
BITRIX_DB_NAME=bitrix
BITRIX_TABLE_PREFIX=b_
```

Либо задайте полный DSN:

```env
BITRIX_DSN=user:password@tcp(127.0.0.1:3306)/bitrix?parseTime=true&charset=utf8mb4
```

### 2. Описание сущности

```go
type Consignee struct {
    ID      int64  `bx:"ID;pk;auto"`
    Address string `bx:"UF_ADDRESS"`
    Inn     string `bx:"UF_INN"`
    Name    string `bx:"UF_NAME"`
    Guid    string `bx:"UF_GUID;ext"` // внешний ID для upsert
}

func (Consignee) TableName() string { return "consignee" }
```

### 3. CRUD

```go
ctx := context.Background()

client, err := bitrixgo.NewFromEnv(ctx)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

repo := bitrixgo.NewRepository[Consignee](client)

id, err := repo.Add(ctx, &Consignee{Name: "ООО Альфа", Guid: "guid-1"})

row, err := repo.GetByID(ctx, id, query.Options{})

rows, err := repo.GetList(ctx, query.Options{
    Filter: filter.Filter{"=UF_NAME": "ООО Альфа"},
    Limit:  10,
})

err = repo.Update(ctx, id, map[string]any{"UF_NAME": "Новое имя"})
err = repo.Delete(ctx, id)
```

## Теги `bx`

| Тег | Описание |
|-----|----------|
| `ID;pk;auto` | Первичный ключ, auto increment |
| `UF_NAME` | Имя колонки в таблице |
| `boolyn` | Bool как `Y`/`N` вместо `1`/`0` |
| `ext` | Внешний идентификатор для upsert (один на struct) |
| `ref=Parent` | FK-поле, ссылается на родителя |
| `rel=Parent;fk=UF_PARENT_ID` | Many-to-one: вложенная struct для preload |
| `rel=Child;fk=UF_PARENT_ID;inverse` | One-to-many: slice дочерних записей |

Bool по умолчанию хранится как `1`/`0`.

## Фильтры

```go
filter.Filter{
    "=UF_ACTIVE":   1,
    "!UF_NAME":     "",
    ">UF_DATE_FROM": time.Now(),
    "%UF_NAME":     "Ромашка",  // LIKE %...%
    "@UF_ID":       []int{1, 2, 3}, // IN (...)
}
```

## Связи

### Many-to-one (дочерняя сторона)

```go
type Contract struct {
    ID           int64      `bx:"ID;pk;auto"`
    ContractorID int64      `bx:"UF_CONTRACTOR_ID;ref=Contractor"`
    Contractor   Contractor `bx:"rel=Contractor;fk=UF_CONTRACTOR_ID"`
    Name         string     `bx:"UF_NAME"`
    Guid         string     `bx:"UF_GUID;ext"`
}

// Preload родителя
contract, _ := contractRepo.GetByID(ctx, id, query.Options{
    With: []string{"Contractor"},
})

// Вставка договора с новым контрагентом
id, _ := contractRepo.AddCascade(ctx, &Contract{
    Name: "Договор №1",
    Contractor: Contractor{Name: "ООО Ромашка", Guid: "c-guid-1"},
})
```

### One-to-many (родительская сторона)

```go
type Contractor struct {
    ID        int64      `bx:"ID;pk;auto"`
    Name      string     `bx:"UF_NAME"`
    Guid      string     `bx:"UF_GUID;ext"`
    Contracts []Contract `bx:"rel=Contract;fk=UF_CONTRACTOR_ID;inverse"`
}

// Preload детей
contractor, _ := contractorRepo.GetByID(ctx, id, query.Options{
    With: []string{"Contracts"},
})

// Импорт: upsert контрагента и договоров по ext
_, _ = contractorRepo.SaveCascade(ctx, &Contractor{
    Guid: "c-guid-1",
    Name: "ООО Ромашка",
    Contracts: []Contract{
        {Guid: "d-guid-1", Name: "Договор №1"},
        {Guid: "d-guid-2", Name: "Договор №2"},
    },
})
```

`SaveCascade` обновляет только переданные дочерние записи; лишние строки в БД **не удаляются**.

### Удаление каскадом

```go
bitrixgo.DeleteCascade[Contractor, Contract](ctx, client, contractorID)
```

## Upsert по внешнему ID

Для данных из внешней системы без PK:

```go
id, err := repo.Upsert(ctx, &Consignee{
    Guid: "external-guid-1",
    Name: "Обновлённое имя",
})
// PK=0 + ext → find by UF_GUID → update или insert
// PK>0       → update по PK

row, err := repo.GetByExt(ctx, "external-guid-1")
```

## API репозитория

| Метод | Описание |
|-------|----------|
| `Add` | INSERT |
| `Update` | UPDATE по PK |
| `Delete` | DELETE по PK |
| `GetByID` | SELECT по PK (+ preload через `query.Options.With`) |
| `GetList` | SELECT с фильтром, сортировкой, limit/offset |
| `GetByExt` | SELECT по ext-колонке |
| `Upsert` | Insert или update по PK / ext |
| `Count` | COUNT с фильтром |
| `AddMulti` / `UpdateMulti` / `DeleteMulti` | Пакетные операции |
| `GetListByFK` | Список дочерних записей по FK |
| `AddCascade` | Insert дочерней записи + upsert родителя |
| `SaveCascade` | Upsert родителя + upsert дочерних (one-to-many) |

## Примеры

```bash
# CRUD и batch-операции
go run ./examples/custom_table

# Many-to-one: Contract → Contractor
go run ./examples/custom_related_many2one

# One-to-many: Contractor + []Contract, re-import по UF_GUID
go run ./examples/custom_related_one2many
```

Перед запуском примеров создайте `.env` из `.env.example`.

## Структура проекта

```
bitrixgo/
  bitrixgo/          # публичный API (Client, NewRepository, DeleteCascade)
    entity/          # meta, scan, preload, relations
    repo/            # Repository, CRUD, batch, upsert, cascade
    filter/          # Bitrix-style фильтры
    query/           # Options для GetList
    errors/          # ErrNotFound, ErrDuplicateExt, ...
  examples/
    custom_table/
    custom_related_many2one/
    custom_related_one2many/
```

## Тесты

```bash
go test ./...
```

## Ограничения

- Только MySQL (прямое подключение к базе Bitrix)
- Preload — отдельные SELECT, без JOIN
- Один уровень cascade за вызов
- `SaveCascade` не синхронизирует удаление «лишних» дочерних записей
- Таблицы должны существовать в базе (миграции не включены)
- CRUD для сущностей не вызывает события bitrix (onBeforeAdd, onAfterUpdate и т.д.)

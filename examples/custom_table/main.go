// Пример использования bitrixgo с кастомной таблицей Bitrix.
//
//	go run ./examples/custom_table
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Fresh4MaN/bx-go/bitrixgo"
	"github.com/Fresh4MaN/bx-go/bitrixgo/filter"
	"github.com/Fresh4MaN/bx-go/bitrixgo/query"
	"github.com/Fresh4MaN/bx-go/bitrixgo/repo"
)

type Consignee struct {
	ID      int64  `bx:"ID;pk;auto"`
	Address string `bx:"UF_ADDRESS"`
	Kpp     string `bx:"UF_KPP"`
	Inn     string `bx:"UF_INN"`
	Name    string `bx:"UF_NAME"`
	Guid    string `bx:"UF_GUID"`
}

func (Consignee) TableName() string { return "consignee" }

func main() {
	ctx := context.Background()

	client, err := bitrixgo.NewFromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	repository := bitrixgo.NewRepository[Consignee](client)

	// --- одиночный CRUD ---
	id, err := repository.Add(ctx, &Consignee{
		Address: "Адрес 1",
		Kpp:     "770101001",
		Inn:     "7707083893",
		Name:    "ООО Альфа",
		Guid:    "guid-1",
	})
	if err != nil {
		log.Fatal("add:", err)
	}
	fmt.Println("added id:", id)

	row, err := repository.GetByID(ctx, id, query.Options{})
	if err != nil {
		log.Fatal("get:", err)
	}
	fmt.Printf("loaded: %+v\n", row)

	rows, err := repository.GetList(ctx, query.Options{
		Filter: filter.Filter{"=UF_ADDRESS": "Адрес"},
		Limit:  10,
	})
	if err != nil {
		log.Fatal("list:", err)
	}
	fmt.Printf("found %d rows\n", len(rows))

	if err := repository.Update(ctx, id, map[string]any{"UF_ADDRESS": "Новый адрес"}); err != nil {
		log.Fatal("update:", err)
	}

	if err := repository.Delete(ctx, id); err != nil {
		log.Fatal("delete:", err)
	}
	fmt.Println("deleted single row")

	// --- AddMulti: пакетная вставка ---
	items := []Consignee{
		{Address: "Адрес 1", Kpp: "770101001", Inn: "7707083893", Name: "ООО Альфа", Guid: "guid-1"},
		{Address: "Адрес 2", Kpp: "770201001", Inn: "7708123456", Name: "ООО Бета", Guid: "guid-2"},
		{Address: "Адрес 3", Kpp: "770301001", Inn: "7709987654", Name: "ООО Гamma", Guid: "guid-3"},
	}

	ids, err := repository.AddMulti(ctx, items)
	if err != nil {
		log.Fatal("add multi:", err)
	}
	fmt.Println("added ids:", ids)

	// --- UpdateMulti: пакетное обновление ---
	if err := repository.UpdateMulti(ctx, []repo.BatchUpdate{
		{ID: ids[0], Fields: map[string]any{"UF_NAME": "ООО Альфа (обновлено)"}},
		{ID: ids[1], Fields: map[string]any{"UF_ADDRESS": "Новый адрес 2", "UF_NAME": "ООО Бета (обновлено)"}},
	}); err != nil {
		log.Fatal("update multi:", err)
	}
	fmt.Println("batch update done")

	rows, err = repository.GetList(ctx, query.Options{Limit: 10})
	if err != nil {
		log.Fatal("list after multi:", err)
	}
	for _, row := range rows {
		fmt.Printf("  id=%d name=%q address=%q\n", row.ID, row.Name, row.Address)
	}

	// --- DeleteMulti: пакетное удаление ---
	idArgs := make([]any, len(ids))
	for i, batchID := range ids {
		idArgs[i] = batchID
	}
	if err := repository.DeleteMulti(ctx, idArgs); err != nil {
		log.Fatal("delete multi:", err)
	}
	fmt.Println("deleted ids:", ids)
}

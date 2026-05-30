// Пример one-to-many: Contractor + []Contract (inverse preload, SaveCascade по ext).
//
//	go run ./examples/custom_related_one2many
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Fresh4MaN/bx-go/bitrixgo"
	"github.com/Fresh4MaN/bx-go/bitrixgo/query"
)

type Contract struct {
	ID           int64      `bx:"ID;pk;auto"`
	ContractorID int64      `bx:"UF_CONTRACTOR_ID;ref=Contractor"`
	Contractor   Contractor `bx:"rel=Contractor;fk=UF_CONTRACTOR_ID"`
	DateTo       time.Time  `bx:"UF_DATE_TO"`
	Name         string     `bx:"UF_NAME"`
	DateFrom     time.Time  `bx:"UF_DATE_FROM"`
	Number       string     `bx:"UF_NUMBER"`
	Guid         string     `bx:"UF_GUID;ext"`
}

func (Contract) TableName() string { return "contract" }

type Contractor struct {
	ID        int64      `bx:"ID;pk;auto"`
	Name      string     `bx:"UF_NAME"`
	ShortName string     `bx:"UF_SHORT_NAME"`
	Guid      string     `bx:"UF_GUID;ext"`
	Contracts []Contract `bx:"rel=Contract;fk=UF_CONTRACTOR_ID;inverse"`
}

func (Contractor) TableName() string { return "contractor" }

func main() {
	ctx := context.Background()

	client, err := bitrixgo.NewFromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	contractorRepo := bitrixgo.NewRepository[Contractor](client)
	now := time.Now()

	contractor := &Contractor{
		Name:      "ООО Ромашка",
		ShortName: "Ромашка",
		Guid:      "contractor-guid-1",
		Contracts: []Contract{
			{
				Name:     "Договор №333",
				Number:   "001/2026",
				Guid:     "contract-guid-1",
				DateFrom: now,
				DateTo:   now.AddDate(1, 0, 0),
			},
			{
				Name:     "Договор №444",
				Number:   "002/2026",
				Guid:     "contract-guid-2",
				DateFrom: now,
				DateTo:   now.AddDate(1, 0, 0),
			},
		},
	}

	// 1. Первый импорт — insert контрагента и двух договоров
	contractorID, err := contractorRepo.SaveCascade(ctx, contractor)
	if err != nil {
		log.Fatal("save cascade:", err)
	}
	fmt.Println("contractor id:", contractorID)

	// 2. Preload договоров
	loaded, err := contractorRepo.GetByExt(ctx, "contractor-guid-1")
	if err != nil {
		log.Fatal("get by ext:", err)
	}
	loadedWithChildren, err := contractorRepo.GetByID(ctx, loaded.ID, query.Options{With: []string{"Contracts"}})
	if err != nil {
		log.Fatal("get with preload:", err)
	}
	fmt.Printf("contractor %q: %d contracts\n", loadedWithChildren.Name, len(loadedWithChildren.Contracts))

	// 3. Повторный импорт — update по ext, без дубликатов
	contractor.Name = "ООО Ромашка (обновлено)"
	contractor.Contracts[0].Name = "Договор №1 (обновлено)"
	contractor.Contracts[1].Name = "Договор №2 (обновлено)"
	if _, err := contractorRepo.SaveCascade(ctx, contractor); err != nil {
		log.Fatal("re-import:", err)
	}
	fmt.Println("re-import ok")

	// 4. Добавить 3-й договор
	contractor.Contracts = append(contractor.Contracts, Contract{
		Name:     "Договор №3",
		Number:   "003/2026",
		Guid:     "contract-guid-3",
		DateFrom: now,
		DateTo:   now.AddDate(1, 0, 0),
	})
	if _, err := contractorRepo.SaveCascade(ctx, contractor); err != nil {
		log.Fatal("add third contract:", err)
	}

	reloaded, err := contractorRepo.GetByID(ctx, contractorID, query.Options{With: []string{"Contracts"}})
	if err != nil {
		log.Fatal("reload:", err)
	}
	fmt.Printf("after 3rd contract: %d contracts\n", len(reloaded.Contracts))

	// 5. Очистка
	if err := bitrixgo.DeleteCascade[Contractor, Contract](ctx, client, contractorID); err != nil {
		log.Fatal("delete cascade:", err)
	}
	fmt.Println("contractor and contracts deleted")
}

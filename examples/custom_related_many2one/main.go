// Пример связанных таблиц Contract → Contractor (many-to-one, FK).
//
//	go run ./examples/custom_related_tables
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
	Active       bool       `bx:"UF_ACTIVE"`
	DateTo       time.Time  `bx:"UF_DATE_TO"`
	Name         string     `bx:"UF_NAME"`
	DateFrom     time.Time  `bx:"UF_DATE_FROM"`
	Number       string     `bx:"UF_NUMBER"`
	Guid         string     `bx:"UF_GUID;ext"`
}

func (Contract) TableName() string { return "contract" }

type Contractor struct {
	ID        int64  `bx:"ID;pk;auto"`
	Active    bool   `bx:"UF_ACTIVE"`
	Name      string `bx:"UF_NAME"`
	ShortName string `bx:"UF_SHORT_NAME"`
	Guid      string `bx:"UF_GUID;ext"`
}

func (Contractor) TableName() string { return "contractor" }

func main() {
	ctx := context.Background()

	client, err := bitrixgo.NewFromEnv(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	contractRepo := bitrixgo.NewRepository[Contract](client)

	now := time.Now()

	// --- AddCascade: контрагент + договор ---
	contractID, err := contractRepo.AddCascade(ctx, &Contract{
		Name:     "Договор №1",
		Number:   "001/2026",
		Guid:     "contract-guid-1",
		Active:   true,
		DateFrom: now,
		DateTo:   now.AddDate(1, 0, 0),
		Contractor: Contractor{
			Name:      "ООО Ромашка",
			ShortName: "Ромашка",
			Guid:      "contractor-guid-1",
			Active:    true,
		},
	})
	if err != nil {
		log.Fatal("add cascade:", err)
	}
	fmt.Println("contract id:", contractID)

	contract, err := contractRepo.GetByID(ctx, contractID, query.Options{With: []string{"Contractor"}})
	if err != nil {
		log.Fatal("get with preload:", err)
	}
	fmt.Printf("contract: %q, contractor: %q (id=%d)\n", contract.Name, contract.Contractor.Name, contract.ContractorID)

	contracts, err := contractRepo.GetListByFK(ctx, "Contractor", contract.ContractorID, query.Options{With: []string{"Contractor"}})
	if err != nil {
		log.Fatal("list by fk:", err)
	}
	fmt.Printf("contracts for contractor %d: %d\n", contract.ContractorID, len(contracts))

	if err := bitrixgo.DeleteCascade[Contractor, Contract](ctx, client, contract.ContractorID); err != nil {
		log.Fatal("delete cascade:", err)
	}
	fmt.Println("contractor and contracts deleted")
}

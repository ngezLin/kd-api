package seeders

import (
	"fmt"
	"kd-api/config"
	"kd-api/models"
	"math/rand"
	"time"
)

// helper untuk pointer string
func ptrString(s string) *string {
	return &s
}

func Seed() {
	rand.Seed(time.Now().UnixNano())

	// ============= Seed Users =============
	users := []models.User{
		{Username: "admin", Password: "admin123", Role: "admin"},
		{Username: "cashier1", Password: "cashier123", Role: "cashier"},
	}

	for _, user := range users {
		config.DB.FirstOrCreate(&user, models.User{Username: user.Username})
	}

	// ============= Seed Items =============
	items := []models.Item{
		{Name: "Indomie Goreng", Description: ptrString("Mie instan rasa original"), Stock: 100, BuyPrice: 2000, Price: 3500, ImageURL: ptrString("https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcRj-aiL5r17BZ2D4HGnRec2dWmTtCrS3Q58yw&s")},
		{Name: "Teh Botol", Description: ptrString("Minuman teh manis dingin"), Stock: 80, BuyPrice: 3000, Price: 5000, ImageURL: ptrString("https://c.alfagift.id/product/1/A6358410001086_A6358410001086_20200423001723369_base.jpg")},
		{Name: "Chitato", Description: ptrString("Keripik kentang sapi panggang"), Stock: 60, BuyPrice: 5000, Price: 8000, ImageURL: ptrString("https://cdn-klik.klikindomaret.com/klik-catalog/product/10001094_meta")},
		{Name: "Aqua Botol", Description: ptrString("Air mineral 600ml"), Stock: 120, BuyPrice: 2500, Price: 4000, ImageURL: ptrString("production.s3.amazonaws.com/media/images/products/2021/06/DSC_0047_copy_TaS0jlu.jpg")},
		{Name: "Silverqueen", Description: ptrString("Coklat almond bar"), Stock: 40, BuyPrice: 10000, Price: 15000, ImageURL: ptrString("https://cdn-klik.klikindomaret.com/klik-catalog/product/10010192_1.jpg")},
		{Name: "Pocari Sweat", Description: ptrString("Minuman isotonik"), Stock: 70, BuyPrice: 4000, Price: 7000, ImageURL: ptrString("https://d2qjkwm11akmwu.cloudfront.net/products/163812_27-5-2022_17-25-30-1665803703.webp")},
		{Name: "Kopiko", Description: ptrString("Permen kopi sachet"), Stock: 200, BuyPrice: 500, Price: 1000, ImageURL: ptrString("https://solvent-production.s3.amazonaws.com/media/images/products/2021/04/2835a.jpg")},
		{Name: "Good Day Coffee", Description: ptrString("Kopi instan sachet"), Stock: 90, BuyPrice: 1500, Price: 2500, ImageURL: ptrString("https://drivethru.klikindomaret.com/twb5/wp-content/uploads/sites/31/2021/07/20045674_1.jpg")},
		{Name: "Roma Biskuit", Description: ptrString("Biskuit kelapa"), Stock: 110, BuyPrice: 3500, Price: 6000, ImageURL: ptrString("https://c.alfagift.id/product/1/1_A10160000601_20220317102027482_base.jpg")},
		{Name: "Oreo", Description: ptrString("Biskuit isi krim"), Stock: 85, BuyPrice: 4000, Price: 7000, ImageURL: ptrString("https://mcgrocer.com/cdn/shop/files/oreo-double-stuff-biscuits-41358218756334.jpg?v=1744908403")},
	}

	for _, item := range items {
		config.DB.FirstOrCreate(&item, models.Item{Name: item.Name})
	}

	// ambil semua items setelah insert
	var allItems []models.Item
	config.DB.Find(&allItems)

	// fungsi helper bikin transaction dengan status tertentu
	createTransaction := func(status string) {
		// pilih 2-3 random items
		n := rand.Intn(2) + 2
		var transItems []models.TransactionItem
		total := float64(0)

		for i := 0; i < n; i++ {
			item := allItems[rand.Intn(len(allItems))]
			qty := rand.Intn(3) + 1
			subtotal := item.Price * float64(qty)
			total += subtotal

			transItems = append(transItems, models.TransactionItem{
				ItemID:   item.ID,
				Quantity: qty,
				Price:    item.Price,
				Subtotal: subtotal,
			})

			if status == "completed" {
				item.Stock -= qty
				config.DB.Save(&item)
			}
			if status == "refunded" {
				config.DB.Save(&item)
			}
		}

		transaction := models.Transaction{
			Status: status,
			Total:  total,
			Items:  transItems,
		}

		// kalau completed, tambahkan payment & change
		if status == "completed" {
			payment := total + float64(rand.Intn(5000)) // random lebih dari total
			change := payment - total
			transaction.Payment = &payment
			transaction.Change = &change
		}

		config.DB.Create(&transaction)
	}

	// ============= Seed Transactions =============
	for i := 0; i < 3; i++ {
		createTransaction("draft")
	}
	for i := 0; i < 3; i++ {
		createTransaction("completed")
	}
	for i := 0; i < 3; i++ {
		createTransaction("refunded")
	}

	fmt.Println("✅ Seeding selesai! 2 users + 10 items + 9 transactions (3 draft, 3 completed, 3 refunded)")
}

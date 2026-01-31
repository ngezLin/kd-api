package response

import "kd-api/models"

// Response khusus untuk role cashier (field dibatasi)
type ItemResponseCashier struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Stock       int     `json:"stock"`
	Price       float64 `json:"price"`
	ImageURL    *string `json:"image_url,omitempty"`
}

// Mapping slice item berdasarkan role user
func FilterItemsForRole(items []models.Item, role string) interface{} {
	if role != "cashier" {
		return items
	}

	result := make([]ItemResponseCashier, len(items))
	for i, item := range items {
		result[i] = mapItemForCashier(item)
	}
	return result
}

// Mapping single item berdasarkan role user
func FilterItemForRole(item models.Item, role string) interface{} {
	if role != "cashier" {
		return item
	}

	return mapItemForCashier(item)
}

// Internal helper, jangan dipakai langsung dari luar
func mapItemForCashier(item models.Item) ItemResponseCashier {
	return ItemResponseCashier{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Stock:       item.Stock,
		Price:       item.Price,
		ImageURL:    item.ImageURL,
	}
}

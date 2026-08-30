package handlers

import (
	"database/sql"
	"net/http"

	"nomorewaste/internal/models"
)

type dashboardStats struct {
	Merchants         int              `json:"merchants"`
	Products          int              `json:"products"`
	TotalStock        int              `json:"total_stock"`
	Tours             int              `json:"tours"`
	VolunteersActive  int              `json:"volunteers_active"`
	VolunteersPending int              `json:"volunteers_pending"`
	Plannings         int              `json:"plannings"`
	Services          int              `json:"services"`
	Collections       int              `json:"collections"`
	ExpiringCount     int              `json:"expiring_count"`
	ExpiredCount      int              `json:"expired_count"`
	ExpiringProducts  []models.Product `json:"expiring_products"`
}

const expiryAlertWindowDays = 3

func (a *App) expiringProducts(limit int) ([]models.Product, error) {
	rows, err := a.DB.Query(`SELECT p.id, p.name, COALESCE(p.category, ''), p.barcode, p.quantity,
		COALESCE(p.expiration_date, ''), COALESCE(c.label, ''), COALESCE(ci.name, ''),
		COALESCE(p.shelf_code, '')
		FROM products p
		LEFT JOIN containers c ON c.id = p.container_id
		LEFT JOIN cities ci ON ci.id = c.city_id
		WHERE p.expiration_date IS NOT NULL AND p.expiration_date != ''
		AND p.quantity > 0
		AND date(p.expiration_date) <= date('now', '+`+itoa(expiryAlertWindowDays)+` day')
		ORDER BY p.expiration_date ASC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Barcode, &p.Quantity, &p.ExpirationDate,
			&p.ContainerName, &p.CityName, &p.ShelfCode); err != nil {
			continue
		}
		p.DaysToExpiry = daysUntil(p.ExpirationDate)
		products = append(products, p)
	}
	return products, nil
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return string(digits)
}

func (a *App) Dashboard(w http.ResponseWriter, r *http.Request) {
	stats := dashboardStats{}
	a.DB.QueryRow("SELECT COUNT(*) FROM merchants").Scan(&stats.Merchants)
	a.DB.QueryRow("SELECT COUNT(*) FROM products").Scan(&stats.Products)
	a.DB.QueryRow("SELECT COALESCE(SUM(quantity), 0) FROM products").Scan(&stats.TotalStock)
	a.DB.QueryRow("SELECT COUNT(*) FROM tours").Scan(&stats.Tours)
	a.DB.QueryRow("SELECT COUNT(*) FROM volunteers WHERE status = 'approved'").Scan(&stats.VolunteersActive)
	a.DB.QueryRow("SELECT COUNT(*) FROM volunteers WHERE status = 'pending'").Scan(&stats.VolunteersPending)
	a.DB.QueryRow("SELECT COUNT(*) FROM plannings").Scan(&stats.Plannings)
	a.DB.QueryRow("SELECT COUNT(*) FROM services").Scan(&stats.Services)
	a.DB.QueryRow("SELECT COUNT(*) FROM collections WHERE status != 'completed'").Scan(&stats.Collections)
	a.DB.QueryRow(`SELECT COUNT(*) FROM products WHERE expiration_date IS NOT NULL AND expiration_date != ''
		AND quantity > 0 AND date(expiration_date) < date('now')`).Scan(&stats.ExpiredCount)
	a.DB.QueryRow(`SELECT COUNT(*) FROM products WHERE expiration_date IS NOT NULL AND expiration_date != ''
		AND quantity > 0 AND date(expiration_date) >= date('now')
		AND date(expiration_date) <= date('now', '+3 day')`).Scan(&stats.ExpiringCount)

	products, err := a.expiringProducts(20)
	if err != nil && err != sql.ErrNoRows {
		products = []models.Product{}
	}
	stats.ExpiringProducts = products
	writeJSON(w, http.StatusOK, stats)
}

func (a *App) ExpiringProducts(w http.ResponseWriter, r *http.Request) {
	products, err := a.expiringProducts(100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, products)
}

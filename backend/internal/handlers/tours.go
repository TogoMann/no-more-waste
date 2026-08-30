package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"nomorewaste/internal/exports"
	"nomorewaste/internal/models"
)

type tourItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type tourRequest struct {
	Label         string            `json:"label"`
	DriverName    string            `json:"driver_name"`
	Destination   string            `json:"destination"`
	ScheduledDate string            `json:"scheduled_date"`
	Status        string            `json:"status"`
	Items         []tourItemRequest `json:"items"`
}

func (a *App) loadTour(id int64) (models.Tour, error) {
	var t models.Tour
	err := a.DB.QueryRow(`SELECT id, label, driver_name, destination, scheduled_date, status, created_at
		FROM tours WHERE id = ?`, id).
		Scan(&t.ID, &t.Label, &t.DriverName, &t.Destination, &t.ScheduledDate, &t.Status, &t.CreatedAt)
	if err != nil {
		return t, err
	}
	rows, err := a.DB.Query(`SELECT ti.id, ti.tour_id, ti.product_id, p.name, ti.quantity
		FROM tour_items ti JOIN products p ON p.id = ti.product_id WHERE ti.tour_id = ?`, id)
	if err != nil {
		return t, err
	}
	defer rows.Close()
	t.Items = []models.TourItem{}
	for rows.Next() {
		var item models.TourItem
		if err := rows.Scan(&item.ID, &item.TourID, &item.ProductID, &item.ProductName, &item.Quantity); err != nil {
			continue
		}
		t.Items = append(t.Items, item)
	}
	return t, nil
}

func (a *App) ListTours(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`SELECT id, label, driver_name, destination, scheduled_date, status, created_at
		FROM tours ORDER BY scheduled_date DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	tours := []models.Tour{}
	for rows.Next() {
		var t models.Tour
		if err := rows.Scan(&t.ID, &t.Label, &t.DriverName, &t.Destination, &t.ScheduledDate, &t.Status, &t.CreatedAt); err != nil {
			continue
		}
		tours = append(tours, t)
	}
	writeJSON(w, http.StatusOK, tours)
}

func (a *App) GetTour(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	tour, err := a.loadTour(id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "tour not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, tour)
}

func (a *App) CreateTour(w http.ResponseWriter, r *http.Request) {
	var req tourRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Label == "" || req.ScheduledDate == "" {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	if isPastDate(req.ScheduledDate) {
		writeError(w, http.StatusBadRequest, "cannot create a tour in the past")
		return
	}
	if req.Status == "" {
		req.Status = "planned"
	}
	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	res, err := tx.Exec(`INSERT INTO tours (label, driver_name, destination, scheduled_date, status)
		VALUES (?, ?, ?, ?, ?)`, req.Label, req.DriverName, req.Destination, req.ScheduledDate, req.Status)
	if err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	tourID, _ := res.LastInsertId()
	for _, item := range req.Items {
		if _, err := tx.Exec(`INSERT INTO tour_items (tour_id, product_id, quantity) VALUES (?, ?, ?)`,
			tourID, item.ProductID, item.Quantity); err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	tour, _ := a.loadTour(tourID)
	writeJSON(w, http.StatusCreated, tour)
}

func (a *App) UpdateTourStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if _, err := a.DB.Exec("UPDATE tours SET status = ? WHERE id = ?", req.Status, id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "tour updated"})
}

func (a *App) DeleteTour(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM tours WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "tour deleted"})
}

func (a *App) TourDeliveryPDF(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	tour, err := a.loadTour(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "tour not found")
		return
	}
	pdfBytes, err := exports.TourDeliveryPDF(tour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "pdf error")
		return
	}
	a.DB.Exec("UPDATE tours SET status = 'delivered' WHERE id = ?", id)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=livraison-tournee-%d.pdf", id))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}

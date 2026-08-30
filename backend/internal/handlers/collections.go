package handlers

import (
	"database/sql"
	"net/http"

	"nomorewaste/internal/auth"
	"nomorewaste/internal/exports"
	"nomorewaste/internal/models"
)

type collectionStopRequest struct {
	MerchantID int64 `json:"merchant_id"`
	OrderIndex int   `json:"order_index"`
}

type collectionRequest struct {
	DriverID      *int64                  `json:"driver_id"`
	Label         string                  `json:"label"`
	ScheduledDate string                  `json:"scheduled_date"`
	Status        string                  `json:"status"`
	Stops         []collectionStopRequest `json:"stops"`
}

type collectedProduct struct {
	Name           string `json:"name"`
	Category       string `json:"category"`
	Barcode        string `json:"barcode"`
	Quantity       int    `json:"quantity"`
	ExpirationDate string `json:"expiration_date"`
	ContainerID    *int64 `json:"container_id"`
	ShelfCode      string `json:"shelf_code"`
	MerchantID     *int64 `json:"merchant_id"`
}

type completeCollectionRequest struct {
	Products []collectedProduct `json:"products"`
}

var allowedCollectionStatuses = map[string]bool{
	"planned":     true,
	"in_progress": true,
	"completed":   true,
}

func (a *App) collectionStops(collectionID int64) []models.CollectionStop {
	rows, err := a.DB.Query(`SELECT cs.id, cs.collection_id, cs.merchant_id, m.company_name,
		COALESCE(m.address, ''), cs.order_index, cs.collected
		FROM collection_stops cs JOIN merchants m ON m.id = cs.merchant_id
		WHERE cs.collection_id = ? ORDER BY cs.order_index`, collectionID)
	if err != nil {
		return []models.CollectionStop{}
	}
	defer rows.Close()
	stops := []models.CollectionStop{}
	for rows.Next() {
		var stop models.CollectionStop
		var collected int
		if err := rows.Scan(&stop.ID, &stop.CollectionID, &stop.MerchantID, &stop.MerchantName,
			&stop.Address, &stop.OrderIndex, &collected); err != nil {
			continue
		}
		stop.Collected = collected > 0
		stops = append(stops, stop)
	}
	return stops
}

func (a *App) loadCollection(id int64) (models.Collection, error) {
	var collection models.Collection
	var completedAt sql.NullString
	err := a.DB.QueryRow(`SELECT c.id, c.driver_id, COALESCE(v.full_name, ''), c.label, c.scheduled_date,
		c.status, c.completed_at, c.created_at
		FROM collections c LEFT JOIN volunteers v ON v.id = c.driver_id WHERE c.id = ?`, id).
		Scan(&collection.ID, &collection.DriverID, &collection.DriverName, &collection.Label,
			&collection.ScheduledDate, &collection.Status, &completedAt, &collection.CreatedAt)
	if err != nil {
		return collection, err
	}
	collection.CompletedAt = completedAt.String
	collection.Stops = a.collectionStops(id)
	collection.StopCount = len(collection.Stops)
	return collection, nil
}

func (a *App) ListCollections(w http.ResponseWriter, r *http.Request) {
	query := `SELECT c.id, c.driver_id, COALESCE(v.full_name, ''), c.label, c.scheduled_date,
		c.status, c.completed_at, c.created_at,
		(SELECT COUNT(*) FROM collection_stops WHERE collection_id = c.id)
		FROM collections c LEFT JOIN volunteers v ON v.id = c.driver_id`
	args := []interface{}{}
	if status := r.URL.Query().Get("status"); status != "" {
		query += " WHERE c.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY c.scheduled_date DESC"
	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	collections := []models.Collection{}
	for rows.Next() {
		var collection models.Collection
		var completedAt sql.NullString
		if err := rows.Scan(&collection.ID, &collection.DriverID, &collection.DriverName, &collection.Label,
			&collection.ScheduledDate, &collection.Status, &completedAt, &collection.CreatedAt,
			&collection.StopCount); err != nil {
			continue
		}
		collection.CompletedAt = completedAt.String
		collections = append(collections, collection)
	}
	writeJSON(w, http.StatusOK, collections)
}

func (a *App) GetCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	collection, err := a.loadCollection(id)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "collection not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, collection)
}

func (a *App) CreateCollection(w http.ResponseWriter, r *http.Request) {
	var req collectionRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Label == "" || req.ScheduledDate == "" {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	if isPastDate(req.ScheduledDate) {
		writeError(w, http.StatusBadRequest, "cannot plan a collection in the past")
		return
	}
	if len(req.Stops) == 0 {
		writeError(w, http.StatusBadRequest, "at least one stop is required")
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	result, err := tx.Exec(`INSERT INTO collections (driver_id, label, scheduled_date, status)
		VALUES (?, ?, ?, 'planned')`, req.DriverID, req.Label, req.ScheduledDate)
	if err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	collectionID, _ := result.LastInsertId()
	for index, stop := range req.Stops {
		orderIndex := stop.OrderIndex
		if orderIndex == 0 {
			orderIndex = index + 1
		}
		if _, err := tx.Exec(`INSERT INTO collection_stops (collection_id, merchant_id, order_index)
			VALUES (?, ?, ?)`, collectionID, stop.MerchantID, orderIndex); err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	collection, _ := a.loadCollection(collectionID)
	writeJSON(w, http.StatusCreated, collection)
}

func (a *App) UpdateCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req collectionRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var currentStatus, currentDate string
	err = a.DB.QueryRow("SELECT status, scheduled_date FROM collections WHERE id = ?", id).
		Scan(&currentStatus, &currentDate)
	if err != nil {
		writeError(w, http.StatusNotFound, "collection not found")
		return
	}
	if currentStatus == "completed" {
		writeError(w, http.StatusBadRequest, "cannot modify a completed collection")
		return
	}
	if isPastDate(currentDate) {
		writeError(w, http.StatusBadRequest, "cannot modify a past collection")
		return
	}
	if isPastDate(req.ScheduledDate) {
		writeError(w, http.StatusBadRequest, "cannot move a collection to the past")
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec(`UPDATE collections SET driver_id = ?, label = ?, scheduled_date = ? WHERE id = ?`,
		req.DriverID, req.Label, req.ScheduledDate, id); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec("DELETE FROM collection_stops WHERE collection_id = ?", id); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	for index, stop := range req.Stops {
		orderIndex := stop.OrderIndex
		if orderIndex == 0 {
			orderIndex = index + 1
		}
		if _, err := tx.Exec(`INSERT INTO collection_stops (collection_id, merchant_id, order_index)
			VALUES (?, ?, ?)`, id, stop.MerchantID, orderIndex); err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	collection, _ := a.loadCollection(id)
	writeJSON(w, http.StatusOK, collection)
}

func (a *App) SetCollectionStatus(w http.ResponseWriter, r *http.Request) {
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
	if !allowedCollectionStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	if req.Status == "completed" {
		writeError(w, http.StatusBadRequest, "use the complete endpoint")
		return
	}
	result, err := a.DB.Exec("UPDATE collections SET status = ? WHERE id = ? AND status != 'completed'",
		req.Status, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeError(w, http.StatusBadRequest, "collection not found or already completed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "collection updated"})
}

func (a *App) DeleteCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM collections WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "collection deleted"})
}

func (a *App) CompleteCollection(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req completeCollectionRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	var currentStatus string
	if err := a.DB.QueryRow("SELECT status FROM collections WHERE id = ?", id).Scan(&currentStatus); err != nil {
		writeError(w, http.StatusNotFound, "collection not found")
		return
	}
	if currentStatus == "completed" {
		writeError(w, http.StatusBadRequest, "collection already completed")
		return
	}
	for _, product := range req.Products {
		if product.Name == "" {
			writeError(w, http.StatusBadRequest, "product name is required")
			return
		}
		if product.ExpirationDate == "" {
			writeError(w, http.StatusBadRequest, "expiration date is required for every product")
			return
		}
		if product.Quantity <= 0 {
			writeError(w, http.StatusBadRequest, "quantity must be positive")
			return
		}
	}
	identity, _ := auth.FromContext(r.Context())
	var createdBy *int64
	if identity != nil {
		createdBy = &identity.UserID
	}
	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	stored := 0
	for _, product := range req.Products {
		barcode := product.Barcode
		if barcode == "" {
			barcode = exports.GenerateBarcodeValue()
		}
		var existingID int64
		var existingQuantity int
		err := tx.QueryRow("SELECT id, quantity FROM products WHERE barcode = ?", barcode).
			Scan(&existingID, &existingQuantity)
		if err == nil {
			if _, err := tx.Exec(`UPDATE products SET quantity = ?, expiration_date = ? WHERE id = ?`,
				existingQuantity+product.Quantity, product.ExpirationDate, existingID); err != nil {
				tx.Rollback()
				writeError(w, http.StatusInternalServerError, "database error")
				return
			}
		} else {
			result, err := tx.Exec(`INSERT INTO products (name, category, barcode, quantity, merchant_id,
				container_id, shelf_code, expiration_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				product.Name, product.Category, barcode, product.Quantity, product.MerchantID,
				product.ContainerID, product.ShelfCode, product.ExpirationDate)
			if err != nil {
				tx.Rollback()
				writeError(w, http.StatusInternalServerError, "database error")
				return
			}
			existingID, _ = result.LastInsertId()
		}
		if _, err := tx.Exec(`INSERT INTO stock_movements (product_id, movement_type, quantity, reason, created_by)
			VALUES (?, 'in', ?, 'Collecte', ?)`, existingID, product.Quantity, createdBy); err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		stored++
	}
	if _, err := tx.Exec(`UPDATE collections SET status = 'completed', completed_at = datetime('now')
		WHERE id = ?`, id); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec("UPDATE collection_stops SET collected = 1 WHERE collection_id = ?", id); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec(`UPDATE donation_offers SET status = 'collected'
		WHERE collection_id = ? AND status = 'scheduled'`, id); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "collection completed",
		"products_stored": stored,
	})
}

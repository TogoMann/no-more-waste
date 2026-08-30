package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nomorewaste/internal/auth"
	"nomorewaste/internal/exports"
	"nomorewaste/internal/models"
)

type productRequest struct {
	Name           string   `json:"name"`
	Category       string   `json:"category"`
	Barcode        string   `json:"barcode"`
	Description    string   `json:"description"`
	Quantity       int      `json:"quantity"`
	MerchantID     *int64   `json:"merchant_id"`
	ContainerID    *int64   `json:"container_id"`
	ShelfCode      string   `json:"shelf_code"`
	ExpirationDate string   `json:"expiration_date"`
	Images         []string `json:"images"`
}

type stockRequest struct {
	MovementType string `json:"movement_type"`
	Quantity     int    `json:"quantity"`
	Reason       string `json:"reason"`
}

func daysUntil(value string) *int {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}
	days := int(parsed.Sub(truncateToday()).Hours() / 24)
	return &days
}

func (a *App) ListProducts(w http.ResponseWriter, r *http.Request) {
	query := `SELECT p.id, p.name, COALESCE(p.category, ''), p.barcode, COALESCE(p.description, ''), p.quantity, p.merchant_id,
		p.container_id, COALESCE(c.label, ''), COALESCE(ci.name, ''), COALESCE(p.shelf_code, ''),
		COALESCE(p.expiration_date, ''), p.created_at,
		(SELECT image FROM product_images WHERE product_id = p.id ORDER BY id LIMIT 1),
		(SELECT COUNT(*) FROM product_images WHERE product_id = p.id)
		FROM products p
		LEFT JOIN containers c ON c.id = p.container_id
		LEFT JOIN cities ci ON ci.id = c.city_id`
	conditions := []string{}
	args := []interface{}{}
	if search := r.URL.Query().Get("search"); search != "" {
		conditions = append(conditions, "(p.name LIKE ? OR p.barcode LIKE ? OR p.category LIKE ? OR p.shelf_code LIKE ?)")
		like := "%" + search + "%"
		args = append(args, like, like, like, like)
	}
	if containerID := r.URL.Query().Get("container_id"); containerID != "" {
		conditions = append(conditions, "p.container_id = ?")
		args = append(args, containerID)
	}
	if cityID := r.URL.Query().Get("city_id"); cityID != "" {
		conditions = append(conditions, "c.city_id = ?")
		args = append(args, cityID)
	}
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY p.id DESC"
	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		var thumbnail sql.NullString
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Barcode, &p.Description, &p.Quantity,
			&p.MerchantID, &p.ContainerID, &p.ContainerName, &p.CityName, &p.ShelfCode,
			&p.ExpirationDate, &p.CreatedAt, &thumbnail, &p.ImageCount); err != nil {
			continue
		}
		p.Thumbnail = thumbnail.String
		p.DaysToExpiry = daysUntil(p.ExpirationDate)
		products = append(products, p)
	}
	writeJSON(w, http.StatusOK, products)
}

func (a *App) productImages(productID int64) []models.ProductImage {
	rows, err := a.DB.Query("SELECT id, image FROM product_images WHERE product_id = ? ORDER BY id", productID)
	if err != nil {
		return []models.ProductImage{}
	}
	defer rows.Close()
	images := []models.ProductImage{}
	for rows.Next() {
		var image models.ProductImage
		if err := rows.Scan(&image.ID, &image.Image); err != nil {
			continue
		}
		images = append(images, image)
	}
	return images
}

func (a *App) GetProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var p models.Product
	err = a.DB.QueryRow(`SELECT p.id, p.name, COALESCE(p.category, ''), p.barcode, COALESCE(p.description, ''), p.quantity, p.merchant_id,
		p.container_id, COALESCE(c.label, ''), COALESCE(ci.name, ''), COALESCE(p.shelf_code, ''),
		COALESCE(p.expiration_date, ''), p.created_at
		FROM products p
		LEFT JOIN containers c ON c.id = p.container_id
		LEFT JOIN cities ci ON ci.id = c.city_id
		WHERE p.id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Category, &p.Barcode, &p.Description, &p.Quantity, &p.MerchantID,
			&p.ContainerID, &p.ContainerName, &p.CityName, &p.ShelfCode, &p.ExpirationDate, &p.CreatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	p.Images = a.productImages(p.ID)
	p.ImageCount = len(p.Images)
	p.DaysToExpiry = daysUntil(p.ExpirationDate)
	writeJSON(w, http.StatusOK, p)
}

func (a *App) GetProductByBarcode(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("barcode")
	var p models.Product
	err := a.DB.QueryRow(`SELECT p.id, p.name, COALESCE(p.category, ''), p.barcode, COALESCE(p.description, ''), p.quantity, p.merchant_id,
		p.container_id, COALESCE(c.label, ''), COALESCE(ci.name, ''), COALESCE(p.shelf_code, ''),
		COALESCE(p.expiration_date, ''), p.created_at
		FROM products p
		LEFT JOIN containers c ON c.id = p.container_id
		LEFT JOIN cities ci ON ci.id = c.city_id
		WHERE p.barcode = ?`, code).
		Scan(&p.ID, &p.Name, &p.Category, &p.Barcode, &p.Description, &p.Quantity, &p.MerchantID,
			&p.ContainerID, &p.ContainerName, &p.CityName, &p.ShelfCode, &p.ExpirationDate, &p.CreatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	p.Images = a.productImages(p.ID)
	p.ImageCount = len(p.Images)
	p.DaysToExpiry = daysUntil(p.ExpirationDate)
	writeJSON(w, http.StatusOK, p)
}

func (a *App) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req productRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "missing name")
		return
	}
	if req.ExpirationDate == "" {
		writeError(w, http.StatusBadRequest, "expiration date is required")
		return
	}
	if req.Barcode == "" {
		req.Barcode = exports.GenerateBarcodeValue()
	}
	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	result, err := tx.Exec(`INSERT INTO products (name, category, barcode, description, quantity, merchant_id,
		container_id, shelf_code, expiration_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, req.Name, req.Category,
		req.Barcode, req.Description, req.Quantity, req.MerchantID, req.ContainerID, req.ShelfCode,
		req.ExpirationDate)
	if err != nil {
		tx.Rollback()
		writeError(w, http.StatusConflict, "barcode already used")
		return
	}
	id, _ := result.LastInsertId()
	for _, image := range req.Images {
		if image == "" {
			continue
		}
		if _, err := tx.Exec("INSERT INTO product_images (product_id, image) VALUES (?, ?)", id, image); err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	p := models.Product{ID: id, Name: req.Name, Category: req.Category, Barcode: req.Barcode,
		Description: req.Description, Quantity: req.Quantity, MerchantID: req.MerchantID,
		ContainerID: req.ContainerID, ShelfCode: req.ShelfCode, ExpirationDate: req.ExpirationDate}
	p.Images = a.productImages(id)
	p.ImageCount = len(p.Images)
	writeJSON(w, http.StatusCreated, p)
}

func (a *App) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req productRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	_, err = a.DB.Exec(`UPDATE products SET name = ?, category = ?, description = ?, merchant_id = ?,
		container_id = ?, shelf_code = ?, expiration_date = COALESCE(NULLIF(?, ''), expiration_date)
		WHERE id = ?`,
		req.Name, req.Category, req.Description, req.MerchantID, req.ContainerID, req.ShelfCode,
		req.ExpirationDate, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	for _, image := range req.Images {
		if image == "" {
			continue
		}
		a.DB.Exec("INSERT INTO product_images (product_id, image) VALUES (?, ?)", id, image)
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "product updated"})
}

func (a *App) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM products WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "product deleted"})
}

func (a *App) DeleteProductImage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	imageID, err := strconv.ParseInt(r.PathValue("imageId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid image id")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM product_images WHERE id = ? AND product_id = ?", imageID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "image deleted"})
}

func (a *App) ProductBarcodeImage(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var code string
	if err := a.DB.QueryRow("SELECT barcode FROM products WHERE id = ?", id).Scan(&code); err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	image, err := exports.BarcodePNGBase64(code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "barcode error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"barcode": code, "image": image})
}

func (a *App) MoveStock(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req stockRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.MovementType != "in" && req.MovementType != "out" {
		writeError(w, http.StatusBadRequest, "invalid movement type")
		return
	}
	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be positive")
		return
	}
	var current int
	if err := a.DB.QueryRow("SELECT quantity FROM products WHERE id = ?", id).Scan(&current); err != nil {
		writeError(w, http.StatusNotFound, "product not found")
		return
	}
	newQuantity := current + req.Quantity
	if req.MovementType == "out" {
		newQuantity = current - req.Quantity
		if newQuantity < 0 {
			writeError(w, http.StatusBadRequest, "insufficient stock")
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
	if _, err := tx.Exec("UPDATE products SET quantity = ? WHERE id = ?", newQuantity, id); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec(`INSERT INTO stock_movements (product_id, movement_type, quantity, reason, created_by)
		VALUES (?, ?, ?, ?, ?)`, id, req.MovementType, req.Quantity, req.Reason, createdBy); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"message": "stock updated", "quantity": newQuantity})
}

func (a *App) ListStockMovements(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := a.DB.Query(`SELECT id, product_id, movement_type, quantity, reason, created_by, created_at
		FROM stock_movements WHERE product_id = ? ORDER BY id DESC`, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	movements := []models.StockMovement{}
	for rows.Next() {
		var m models.StockMovement
		if err := rows.Scan(&m.ID, &m.ProductID, &m.MovementType, &m.Quantity, &m.Reason, &m.CreatedBy, &m.CreatedAt); err != nil {
			continue
		}
		movements = append(movements, m)
	}
	writeJSON(w, http.StatusOK, movements)
}

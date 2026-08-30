package handlers

import (
	"database/sql"
	"net/http"

	"nomorewaste/internal/models"
)

type containerRequest struct {
	CityID   int64  `json:"city_id"`
	Label    string `json:"label"`
	Address  string `json:"address"`
	Capacity int    `json:"capacity"`
	Status   string `json:"status"`
}

func (a *App) ListCities(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query("SELECT id, name FROM cities ORDER BY name")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	cities := []models.City{}
	for rows.Next() {
		var city models.City
		if err := rows.Scan(&city.ID, &city.Name); err != nil {
			continue
		}
		cities = append(cities, city)
	}
	writeJSON(w, http.StatusOK, cities)
}

func (a *App) ListContainers(w http.ResponseWriter, r *http.Request) {
	query := `SELECT c.id, c.city_id, ci.name, c.label, c.address, c.capacity, c.status, c.created_at,
		COALESCE((SELECT SUM(p.quantity) FROM products p WHERE p.container_id = c.id), 0),
		(SELECT COUNT(*) FROM products p WHERE p.container_id = c.id)
		FROM containers c JOIN cities ci ON ci.id = c.city_id`
	args := []interface{}{}
	if cityID := r.URL.Query().Get("city_id"); cityID != "" {
		query += " WHERE c.city_id = ?"
		args = append(args, cityID)
	}
	query += " ORDER BY ci.name, c.label"
	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	containers := []models.Container{}
	for rows.Next() {
		var c models.Container
		if err := rows.Scan(&c.ID, &c.CityID, &c.CityName, &c.Label, &c.Address, &c.Capacity,
			&c.Status, &c.CreatedAt, &c.Stored, &c.Products); err != nil {
			continue
		}
		if c.Capacity > 0 {
			c.Occupancy = c.Stored * 100 / c.Capacity
		}
		containers = append(containers, c)
	}
	writeJSON(w, http.StatusOK, containers)
}

func (a *App) GetContainer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var c models.Container
	err = a.DB.QueryRow(`SELECT c.id, c.city_id, ci.name, c.label, c.address, c.capacity, c.status, c.created_at,
		COALESCE((SELECT SUM(p.quantity) FROM products p WHERE p.container_id = c.id), 0),
		(SELECT COUNT(*) FROM products p WHERE p.container_id = c.id)
		FROM containers c JOIN cities ci ON ci.id = c.city_id WHERE c.id = ?`, id).
		Scan(&c.ID, &c.CityID, &c.CityName, &c.Label, &c.Address, &c.Capacity, &c.Status,
			&c.CreatedAt, &c.Stored, &c.Products)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "container not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if c.Capacity > 0 {
		c.Occupancy = c.Stored * 100 / c.Capacity
	}
	writeJSON(w, http.StatusOK, c)
}

func (a *App) CreateContainer(w http.ResponseWriter, r *http.Request) {
	var req containerRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Label == "" || req.CityID == 0 {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	if req.Capacity <= 0 {
		req.Capacity = 100
	}
	if req.Status == "" {
		req.Status = "active"
	}
	result, err := a.DB.Exec(`INSERT INTO containers (city_id, label, address, capacity, status)
		VALUES (?, ?, ?, ?, ?)`, req.CityID, req.Label, req.Address, req.Capacity, req.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	id, _ := result.LastInsertId()
	writeJSON(w, http.StatusCreated, models.Container{ID: id, CityID: req.CityID, Label: req.Label,
		Address: req.Address, Capacity: req.Capacity, Status: req.Status})
}

func (a *App) UpdateContainer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req containerRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Label == "" || req.CityID == 0 {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	result, err := a.DB.Exec(`UPDATE containers SET city_id = ?, label = ?, address = ?, capacity = ?, status = ?
		WHERE id = ?`, req.CityID, req.Label, req.Address, req.Capacity, req.Status, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeError(w, http.StatusNotFound, "container not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "container updated"})
}

func (a *App) DeleteContainer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM containers WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "container deleted"})
}

func (a *App) ContainerProducts(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	rows, err := a.DB.Query(`SELECT p.id, p.name, COALESCE(p.category, ''), p.barcode, COALESCE(p.description, ''), p.quantity,
		COALESCE(p.shelf_code, ''),
		(SELECT image FROM product_images WHERE product_id = p.id ORDER BY id LIMIT 1)
		FROM products p WHERE p.container_id = ? ORDER BY p.shelf_code, p.name`, id)
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
			&p.ShelfCode, &thumbnail); err != nil {
			continue
		}
		p.Thumbnail = thumbnail.String
		products = append(products, p)
	}
	writeJSON(w, http.StatusOK, products)
}

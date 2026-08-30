package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"nomorewaste/internal/auth"
	"nomorewaste/internal/models"
)

var allowedServiceCategories = map[string]bool{
	"cuisine":     true,
	"bricolage":   true,
	"electricite": true,
	"plomberie":   true,
	"reparation":  true,
	"vehicule":    true,
	"gardiennage": true,
}

type serviceRequest struct {
	Title       string `json:"title"`
	Category    string `json:"category"`
	Description string `json:"description"`
	DateTime    string `json:"date_time"`
	Location    string `json:"location"`
	MaxCapacity int    `json:"max_capacity"`
	Status      string `json:"status"`
}

func (a *App) membershipIsValid(userID int64) bool {
	var hasPaid int
	var endDate sql.NullString
	err := a.DB.QueryRow("SELECT has_paid_dues, membership_end_date FROM users WHERE id = ?", userID).
		Scan(&hasPaid, &endDate)
	if err != nil || hasPaid == 0 || !endDate.Valid || endDate.String == "" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", endDate.String)
	if err != nil {
		return false
	}
	return !parsed.Before(truncateToday())
}

func truncateToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func isPastDateTime(value string) bool {
	if len(value) >= 10 {
		return isPastDate(value[:10])
	}
	return false
}

func (a *App) ListServices(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	var viewerID int64
	if identity != nil {
		viewerID = identity.UserID
	}
	query := `SELECT s.id, s.title, s.category, COALESCE(s.description, ''), s.date_time,
		COALESCE(s.location, ''), s.max_capacity, s.status, s.created_at,
		(SELECT COUNT(*) FROM service_subscriptions WHERE service_id = s.id),
		(SELECT COUNT(*) FROM service_subscriptions WHERE service_id = s.id AND user_id = ?)
		FROM services s`
	args := []interface{}{viewerID}
	if category := r.URL.Query().Get("category"); category != "" {
		query += " WHERE s.category = ?"
		args = append(args, category)
	}
	query += " ORDER BY s.date_time"
	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	services := []models.Service{}
	for rows.Next() {
		var service models.Service
		var subscribed int
		if err := rows.Scan(&service.ID, &service.Title, &service.Category, &service.Description,
			&service.DateTime, &service.Location, &service.MaxCapacity, &service.Status,
			&service.CreatedAt, &service.SubscriberCount, &subscribed); err != nil {
			continue
		}
		service.Subscribed = subscribed > 0
		services = append(services, service)
	}
	writeJSON(w, http.StatusOK, services)
}

func (a *App) serviceSubscribers(serviceID int64) []models.Participant {
	rows, err := a.DB.Query(`SELECT u.id, u.full_name, u.email, ss.subscribed_at
		FROM service_subscriptions ss JOIN users u ON u.id = ss.user_id
		WHERE ss.service_id = ? ORDER BY ss.subscribed_at`, serviceID)
	if err != nil {
		return []models.Participant{}
	}
	defer rows.Close()
	subscribers := []models.Participant{}
	for rows.Next() {
		var participant models.Participant
		if err := rows.Scan(&participant.UserID, &participant.FullName, &participant.Email,
			&participant.JoinedAt); err != nil {
			continue
		}
		subscribers = append(subscribers, participant)
	}
	return subscribers
}

func (a *App) GetService(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	identity, _ := auth.FromContext(r.Context())
	var viewerID int64
	if identity != nil {
		viewerID = identity.UserID
	}
	var service models.Service
	var subscribed int
	err = a.DB.QueryRow(`SELECT s.id, s.title, s.category, COALESCE(s.description, ''), s.date_time,
		COALESCE(s.location, ''), s.max_capacity, s.status, s.created_at,
		(SELECT COUNT(*) FROM service_subscriptions WHERE service_id = s.id),
		(SELECT COUNT(*) FROM service_subscriptions WHERE service_id = s.id AND user_id = ?)
		FROM services s WHERE s.id = ?`, viewerID, id).
		Scan(&service.ID, &service.Title, &service.Category, &service.Description, &service.DateTime,
			&service.Location, &service.MaxCapacity, &service.Status, &service.CreatedAt,
			&service.SubscriberCount, &subscribed)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	service.Subscribed = subscribed > 0
	if identity != nil && identity.Role == "admin" {
		service.Subscribers = a.serviceSubscribers(id)
	}
	writeJSON(w, http.StatusOK, service)
}

func (a *App) CreateService(w http.ResponseWriter, r *http.Request) {
	var req serviceRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Title == "" || req.DateTime == "" {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	if !allowedServiceCategories[req.Category] {
		writeError(w, http.StatusBadRequest, "invalid category")
		return
	}
	if isPastDateTime(req.DateTime) {
		writeError(w, http.StatusBadRequest, "cannot create a service in the past")
		return
	}
	if req.MaxCapacity <= 0 {
		req.MaxCapacity = 10
	}
	if req.Status == "" {
		req.Status = "open"
	}
	result, err := a.DB.Exec(`INSERT INTO services (title, category, description, date_time, location,
		max_capacity, status) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.Title, req.Category, req.Description, req.DateTime, req.Location, req.MaxCapacity, req.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	id, _ := result.LastInsertId()
	writeJSON(w, http.StatusCreated, models.Service{ID: id, Title: req.Title, Category: req.Category,
		Description: req.Description, DateTime: req.DateTime, Location: req.Location,
		MaxCapacity: req.MaxCapacity, Status: req.Status})
}

func (a *App) UpdateService(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req serviceRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Title == "" || req.DateTime == "" {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	if !allowedServiceCategories[req.Category] {
		writeError(w, http.StatusBadRequest, "invalid category")
		return
	}
	var currentDate string
	if err := a.DB.QueryRow("SELECT date_time FROM services WHERE id = ?", id).Scan(&currentDate); err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if isPastDateTime(currentDate) {
		writeError(w, http.StatusBadRequest, "cannot modify a past service")
		return
	}
	if isPastDateTime(req.DateTime) {
		writeError(w, http.StatusBadRequest, "cannot move a service to the past")
		return
	}
	var subscriberCount int
	a.DB.QueryRow("SELECT COUNT(*) FROM service_subscriptions WHERE service_id = ?", id).Scan(&subscriberCount)
	if req.MaxCapacity < subscriberCount {
		writeError(w, http.StatusBadRequest, "capacity below current subscribers")
		return
	}
	if req.Status == "" {
		req.Status = "open"
	}
	_, err = a.DB.Exec(`UPDATE services SET title = ?, category = ?, description = ?, date_time = ?,
		location = ?, max_capacity = ?, status = ? WHERE id = ?`,
		req.Title, req.Category, req.Description, req.DateTime, req.Location, req.MaxCapacity, req.Status, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "service updated"})
}

func (a *App) DeleteService(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM services WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "service deleted"})
}

func (a *App) SubscribeService(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !a.membershipIsValid(identity.UserID) {
		writeError(w, http.StatusPaymentRequired, "membership dues not paid")
		return
	}
	var dateTime, status string
	var maxCapacity int
	err = a.DB.QueryRow("SELECT date_time, status, max_capacity FROM services WHERE id = ?", id).
		Scan(&dateTime, &status, &maxCapacity)
	if err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if status != "open" {
		writeError(w, http.StatusConflict, "service is closed")
		return
	}
	if isPastDateTime(dateTime) {
		writeError(w, http.StatusBadRequest, "service already passed")
		return
	}
	var count int
	a.DB.QueryRow("SELECT COUNT(*) FROM service_subscriptions WHERE service_id = ?", id).Scan(&count)
	if count >= maxCapacity {
		writeError(w, http.StatusConflict, "service is full")
		return
	}
	if _, err := a.DB.Exec("INSERT OR IGNORE INTO service_subscriptions (service_id, user_id) VALUES (?, ?)",
		id, identity.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "subscribed"})
}

func (a *App) UnsubscribeService(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var dateTime string
	if err := a.DB.QueryRow("SELECT date_time FROM services WHERE id = ?", id).Scan(&dateTime); err != nil {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	if isPastDateTime(dateTime) {
		writeError(w, http.StatusBadRequest, "service already passed")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM service_subscriptions WHERE service_id = ? AND user_id = ?",
		id, identity.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "unsubscribed"})
}

func (a *App) MyServices(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := a.DB.Query(`SELECT s.id, s.title, s.category, COALESCE(s.description, ''), s.date_time,
		COALESCE(s.location, ''), s.max_capacity, s.status, s.created_at,
		(SELECT COUNT(*) FROM service_subscriptions WHERE service_id = s.id)
		FROM services s JOIN service_subscriptions ss ON ss.service_id = s.id
		WHERE ss.user_id = ? ORDER BY s.date_time`, identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	services := []models.Service{}
	for rows.Next() {
		var service models.Service
		if err := rows.Scan(&service.ID, &service.Title, &service.Category, &service.Description,
			&service.DateTime, &service.Location, &service.MaxCapacity, &service.Status,
			&service.CreatedAt, &service.SubscriberCount); err != nil {
			continue
		}
		service.Subscribed = true
		services = append(services, service)
	}
	writeJSON(w, http.StatusOK, services)
}

package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"nomorewaste/internal/auth"
	"nomorewaste/internal/models"
)

var allowedDonationTypes = map[string]bool{"food": true, "object": true}

type donationRequest struct {
	Title          string `json:"title"`
	DonationType   string `json:"donation_type"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	ExpirationDate string `json:"expiration_date"`
	PickupAddress  string `json:"pickup_address"`
	AvailableFrom  string `json:"available_from"`
}

type donationReviewRequest struct {
	Status        string `json:"status"`
	ReviewNote    string `json:"review_note"`
	ScheduledDate string `json:"scheduled_date"`
	DriverID      *int64 `json:"driver_id"`
	CollectionID  *int64 `json:"collection_id"`
}

func (a *App) scanDonations(rows *sql.Rows) []models.DonationOffer {
	defer rows.Close()
	offers := []models.DonationOffer{}
	for rows.Next() {
		var offer models.DonationOffer
		var expiration, pickup, available, note, category, description, collectionDate sql.NullString
		if err := rows.Scan(&offer.ID, &offer.UserID, &offer.MerchantID, &offer.Title, &offer.DonationType,
			&category, &description, &offer.Quantity, &expiration, &pickup, &available,
			&offer.Status, &note, &offer.CollectionID, &offer.CreatedAt, &offer.DonorName,
			&offer.CompanyName, &collectionDate); err != nil {
			continue
		}
		offer.Category = category.String
		offer.Description = description.String
		offer.ExpirationDate = expiration.String
		offer.PickupAddress = pickup.String
		offer.AvailableFrom = available.String
		offer.ReviewNote = note.String
		offer.CollectionDate = collectionDate.String
		offers = append(offers, offer)
	}
	return offers
}

const donationSelect = `SELECT d.id, d.user_id, d.merchant_id, d.title, d.donation_type, d.category,
	d.description, d.quantity, d.expiration_date, d.pickup_address, d.available_from, d.status,
	d.review_note, d.collection_id, d.created_at, u.full_name, COALESCE(u.company_name, ''),
	COALESCE(c.scheduled_date, '')
	FROM donation_offers d
	JOIN users u ON u.id = d.user_id
	LEFT JOIN collections c ON c.id = d.collection_id`

func (a *App) ListDonations(w http.ResponseWriter, r *http.Request) {
	query := donationSelect
	args := []interface{}{}
	if status := r.URL.Query().Get("status"); status != "" {
		query += " WHERE d.status = ?"
		args = append(args, status)
	}
	query += " ORDER BY d.created_at DESC"
	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, a.scanDonations(rows))
}

func (a *App) MyDonations(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := a.DB.Query(donationSelect+" WHERE d.user_id = ? ORDER BY d.created_at DESC", identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, a.scanDonations(rows))
}

func (a *App) CreateDonation(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req donationRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "le titre est obligatoire")
		return
	}
	if !allowedDonationTypes[req.DonationType] {
		req.DonationType = "food"
	}
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	if req.DonationType == "food" && req.ExpirationDate == "" {
		writeError(w, http.StatusBadRequest, "la DLC est obligatoire pour un don alimentaire")
		return
	}
	if req.ExpirationDate != "" && isPastDate(req.ExpirationDate) {
		writeError(w, http.StatusBadRequest, "la DLC ne peut pas etre dans le passe")
		return
	}
	if req.AvailableFrom != "" && isPastDate(req.AvailableFrom) {
		writeError(w, http.StatusBadRequest, "la date de mise a disposition ne peut pas etre dans le passe")
		return
	}
	if req.PickupAddress == "" {
		a.DB.QueryRow("SELECT COALESCE(address, '') FROM users WHERE id = ?", identity.UserID).
			Scan(&req.PickupAddress)
	}
	var merchantID interface{}
	var resolved int64
	if err := a.DB.QueryRow("SELECT id FROM merchants WHERE user_id = ?", identity.UserID).
		Scan(&resolved); err == nil {
		merchantID = resolved
	}
	result, err := a.DB.Exec(`INSERT INTO donation_offers (user_id, merchant_id, title, donation_type,
		category, description, quantity, expiration_date, pickup_address, available_from, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		identity.UserID, merchantID, req.Title, req.DonationType, req.Category, req.Description,
		req.Quantity, req.ExpirationDate, req.PickupAddress, req.AvailableFrom)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	id, _ := result.LastInsertId()
	writeJSON(w, http.StatusCreated, models.DonationOffer{ID: id, UserID: identity.UserID,
		Title: req.Title, DonationType: req.DonationType, Category: req.Category,
		Description: req.Description, Quantity: req.Quantity, ExpirationDate: req.ExpirationDate,
		PickupAddress: req.PickupAddress, AvailableFrom: req.AvailableFrom, Status: "pending"})
}

func (a *App) DeleteDonation(w http.ResponseWriter, r *http.Request) {
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
	var ownerID int64
	var status string
	if err := a.DB.QueryRow("SELECT user_id, status FROM donation_offers WHERE id = ?", id).
		Scan(&ownerID, &status); err != nil {
		writeError(w, http.StatusNotFound, "don introuvable")
		return
	}
	if identity.Role != "admin" && ownerID != identity.UserID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	if status == "scheduled" || status == "collected" {
		writeError(w, http.StatusBadRequest, "ce don est deja planifie")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM donation_offers WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "don supprime"})
}

func (a *App) ReviewDonation(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	identity, _ := auth.FromContext(r.Context())
	var req donationReviewRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Status != "approved" && req.Status != "rejected" {
		writeError(w, http.StatusBadRequest, "statut invalide")
		return
	}
	var offer models.DonationOffer
	var merchantID sql.NullInt64
	var pickup sql.NullString
	var currentStatus string
	err = a.DB.QueryRow(`SELECT id, user_id, merchant_id, title, COALESCE(pickup_address, ''), status
		FROM donation_offers WHERE id = ?`, id).
		Scan(&offer.ID, &offer.UserID, &merchantID, &offer.Title, &pickup, &currentStatus)
	if err != nil {
		writeError(w, http.StatusNotFound, "don introuvable")
		return
	}
	if currentStatus != "pending" {
		writeError(w, http.StatusBadRequest, "ce don a deja ete traite")
		return
	}

	var reviewerID interface{}
	if identity != nil {
		reviewerID = identity.UserID
	}

	if req.Status == "rejected" {
		if _, err := a.DB.Exec(`UPDATE donation_offers SET status = 'rejected', review_note = ?,
			reviewed_by = ? WHERE id = ?`, req.ReviewNote, reviewerID, id); err != nil {
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "don refuse"})
		return
	}

	if !merchantID.Valid {
		writeError(w, http.StatusBadRequest, "ce donateur n est pas rattache a une fiche commercant")
		return
	}
	scheduledDate := req.ScheduledDate
	if scheduledDate == "" {
		scheduledDate = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}
	if isPastDate(scheduledDate) {
		writeError(w, http.StatusBadRequest, "la collecte ne peut pas etre planifiee dans le passe")
		return
	}

	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	var collectionID int64
	if req.CollectionID != nil && *req.CollectionID > 0 {
		collectionID = *req.CollectionID
		var existingStatus string
		if err := tx.QueryRow("SELECT status FROM collections WHERE id = ?", collectionID).
			Scan(&existingStatus); err != nil {
			tx.Rollback()
			writeError(w, http.StatusBadRequest, "tournee introuvable")
			return
		}
		if existingStatus == "completed" {
			tx.Rollback()
			writeError(w, http.StatusBadRequest, "cette tournee est deja terminee")
			return
		}
	} else {
		label := "Ramassage don - " + offer.Title
		result, err := tx.Exec(`INSERT INTO collections (driver_id, label, scheduled_date, status)
			VALUES (?, ?, ?, 'planned')`, req.DriverID, label, scheduledDate)
		if err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
		collectionID, _ = result.LastInsertId()
	}

	var alreadyStop int
	tx.QueryRow("SELECT COUNT(*) FROM collection_stops WHERE collection_id = ? AND merchant_id = ?",
		collectionID, merchantID.Int64).Scan(&alreadyStop)
	if alreadyStop == 0 {
		var nextIndex int
		tx.QueryRow("SELECT COALESCE(MAX(order_index), 0) + 1 FROM collection_stops WHERE collection_id = ?",
			collectionID).Scan(&nextIndex)
		if _, err := tx.Exec(`INSERT INTO collection_stops (collection_id, merchant_id, order_index)
			VALUES (?, ?, ?)`, collectionID, merchantID.Int64, nextIndex); err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}
	if _, err := tx.Exec(`UPDATE donation_offers SET status = 'scheduled', review_note = ?,
		reviewed_by = ?, collection_id = ? WHERE id = ?`,
		req.ReviewNote, reviewerID, collectionID, id); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":        "don valide, collecte planifiee",
		"collection_id":  collectionID,
		"scheduled_date": scheduledDate,
	})
}

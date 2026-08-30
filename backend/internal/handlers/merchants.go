package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"nomorewaste/internal/models"
)

type merchantRequest struct {
	CompanyName     string `json:"company_name"`
	ContactName     string `json:"contact_name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Address         string `json:"address"`
	MembershipStart string `json:"membership_start"`
	MembershipEnd   string `json:"membership_end"`
	Status          string `json:"status"`
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func (a *App) ListMerchants(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query(`SELECT id, user_id, company_name, contact_name, email, phone, address,
		membership_start, membership_end, status, created_at FROM merchants ORDER BY id DESC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	merchants := []models.Merchant{}
	for rows.Next() {
		var m models.Merchant
		if err := rows.Scan(&m.ID, &m.UserID, &m.CompanyName, &m.ContactName, &m.Email, &m.Phone,
			&m.Address, &m.MembershipStart, &m.MembershipEnd, &m.Status, &m.CreatedAt); err != nil {
			continue
		}
		merchants = append(merchants, m)
	}
	writeJSON(w, http.StatusOK, merchants)
}

func (a *App) GetMerchant(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var m models.Merchant
	err = a.DB.QueryRow(`SELECT id, user_id, company_name, contact_name, email, phone, address,
		membership_start, membership_end, status, created_at FROM merchants WHERE id = ?`, id).
		Scan(&m.ID, &m.UserID, &m.CompanyName, &m.ContactName, &m.Email, &m.Phone,
			&m.Address, &m.MembershipStart, &m.MembershipEnd, &m.Status, &m.CreatedAt)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "merchant not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (a *App) CreateMerchant(w http.ResponseWriter, r *http.Request) {
	var req merchantRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.CompanyName == "" || req.ContactName == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	if req.MembershipStart == "" {
		req.MembershipStart = time.Now().Format("2006-01-02")
	}
	if req.MembershipEnd == "" {
		start, _ := time.Parse("2006-01-02", req.MembershipStart)
		req.MembershipEnd = start.AddDate(1, 0, 0).Format("2006-01-02")
	}
	if req.Status == "" {
		req.Status = "active"
	}
	res, err := a.DB.Exec(`INSERT INTO merchants (company_name, contact_name, email, phone, address,
		membership_start, membership_end, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.CompanyName, req.ContactName, req.Email, req.Phone, req.Address,
		req.MembershipStart, req.MembershipEnd, req.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	id, _ := res.LastInsertId()
	req2 := models.Merchant{ID: id, CompanyName: req.CompanyName, ContactName: req.ContactName,
		Email: req.Email, Phone: req.Phone, Address: req.Address,
		MembershipStart: req.MembershipStart, MembershipEnd: req.MembershipEnd, Status: req.Status}
	writeJSON(w, http.StatusCreated, req2)
}

func (a *App) UpdateMerchant(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req merchantRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	_, err = a.DB.Exec(`UPDATE merchants SET company_name = ?, contact_name = ?, email = ?, phone = ?,
		address = ?, membership_start = ?, membership_end = ?, status = ? WHERE id = ?`,
		req.CompanyName, req.ContactName, req.Email, req.Phone, req.Address,
		req.MembershipStart, req.MembershipEnd, req.Status, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "merchant updated"})
}

func (a *App) DeleteMerchant(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM merchants WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "merchant deleted"})
}

type renewalNotice struct {
	MerchantID    int64  `json:"merchant_id"`
	CompanyName   string `json:"company_name"`
	Email         string `json:"email"`
	MembershipEnd string `json:"membership_end"`
	DaysLeft      int    `json:"days_left"`
	Message       string `json:"message"`
}

func (a *App) MembershipReminders(w http.ResponseWriter, r *http.Request) {
	windowDays := 30
	if q := r.URL.Query().Get("days"); q != "" {
		if parsed, err := strconv.Atoi(q); err == nil {
			windowDays = parsed
		}
	}
	rows, err := a.DB.Query(`SELECT id, company_name, email, membership_end FROM merchants
		WHERE status = 'active' ORDER BY membership_end ASC`)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	notices := []renewalNotice{}
	now := time.Now()
	for rows.Next() {
		var id int64
		var company, email, end string
		if err := rows.Scan(&id, &company, &email, &end); err != nil {
			continue
		}
		endDate, err := time.Parse("2006-01-02", end)
		if err != nil {
			continue
		}
		daysLeft := int(endDate.Sub(now).Hours() / 24)
		if daysLeft <= windowDays {
			message := "Renouvellement à prévoir"
			if daysLeft < 0 {
				message = "Adhésion expirée"
			}
			notices = append(notices, renewalNotice{
				MerchantID: id, CompanyName: company, Email: email,
				MembershipEnd: end, DaysLeft: daysLeft, Message: message,
			})
		}
	}
	writeJSON(w, http.StatusOK, notices)
}

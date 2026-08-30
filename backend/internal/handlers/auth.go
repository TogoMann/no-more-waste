package handlers

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"nomorewaste/internal/auth"
	"nomorewaste/internal/models"
	"nomorewaste/internal/siret"
)

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	FullName    string `json:"full_name"`
	Role        string `json:"role"`
	CompanyName string `json:"company_name"`
	Siret       string `json:"siret"`
	Phone       string `json:"phone"`
	Address     string `json:"address"`
	CityID      *int64 `json:"city_id"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

var allowedRegisterRoles = map[string]bool{
	"member":    true,
	"merchant":  true,
	"volunteer": true,
}

func (a *App) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if req.Email == "" || req.Password == "" || req.FullName == "" {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	if req.Role == "" || !allowedRegisterRoles[req.Role] {
		req.Role = "member"
	}
	normalizedSiret := ""
	if req.Role == "merchant" {
		req.CompanyName = strings.TrimSpace(req.CompanyName)
		if req.CompanyName == "" {
			writeError(w, http.StatusBadRequest, "le nom de l entreprise est obligatoire")
			return
		}
		company, err := siret.Lookup(req.Siret)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if company.Verified && !company.Active {
			writeError(w, http.StatusBadRequest, "cet etablissement est ferme")
			return
		}
		normalizedSiret = company.Siret
		var existing int
		a.DB.QueryRow("SELECT COUNT(*) FROM users WHERE siret = ?", normalizedSiret).Scan(&existing)
		if existing > 0 {
			writeError(w, http.StatusConflict, "ce SIRET est deja enregistre")
			return
		}
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash error")
		return
	}
	var siretValue interface{}
	var companyValue interface{}
	if normalizedSiret != "" {
		siretValue = normalizedSiret
		companyValue = req.CompanyName
	}
	res, err := a.DB.Exec(`INSERT INTO users (email, password_hash, full_name, role, company_name, siret,
		phone, address, city_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Email, hash, req.FullName, req.Role, companyValue, siretValue, req.Phone, req.Address, req.CityID)
	if err != nil {
		writeError(w, http.StatusConflict, "email already used")
		return
	}
	id, _ := res.LastInsertId()
	if req.Role == "merchant" {
		endDate := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
		a.DB.Exec(`INSERT INTO merchants (user_id, company_name, contact_name, email, phone, address,
			membership_end, status) VALUES (?, ?, ?, ?, ?, ?, ?, 'active')`,
			id, req.CompanyName, req.FullName, req.Email, req.Phone, req.Address, endDate)
	}
	user := models.User{ID: id, Email: req.Email, FullName: req.FullName, Role: req.Role, Status: "active",
		CompanyName: req.CompanyName, Siret: normalizedSiret}
	token, err := auth.GenerateToken(a.JWTSecret, id, req.Email, req.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token error")
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

func (a *App) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	var user models.User
	err := a.DB.QueryRow(
		"SELECT id, email, password_hash, full_name, role, status FROM users WHERE email = ?",
		req.Email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Role, &user.Status)
	if err == sql.ErrNoRows || !auth.CheckPassword(req.Password, user.PasswordHash) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if user.Status == "banned" {
		writeError(w, http.StatusForbidden, "account suspended")
		return
	}
	token, err := auth.GenerateToken(a.JWTSecret, user.ID, user.Email, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token error")
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

func (a *App) Me(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var user models.User
	err := a.DB.QueryRow(
		"SELECT id, email, full_name, role, status, created_at FROM users WHERE id = ?",
		identity.UserID,
	).Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.Status, &user.CreatedAt)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (a *App) ListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query("SELECT id, email, full_name, role, status, created_at FROM users ORDER BY id DESC")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	users := []models.User{}
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.Status, &user.CreatedAt); err != nil {
			continue
		}
		users = append(users, user)
	}
	writeJSON(w, http.StatusOK, users)
}

func (a *App) VerifySiret(w http.ResponseWriter, r *http.Request) {
	value := r.URL.Query().Get("siret")
	if value == "" {
		writeError(w, http.StatusBadRequest, "parametre siret manquant")
		return
	}
	company, err := siret.Lookup(value)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var existing int
	a.DB.QueryRow("SELECT COUNT(*) FROM users WHERE siret = ?", company.Siret).Scan(&existing)
	if existing > 0 {
		writeError(w, http.StatusConflict, "ce SIRET est deja enregistre")
		return
	}
	writeJSON(w, http.StatusOK, company)
}

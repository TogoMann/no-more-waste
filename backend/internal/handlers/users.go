package handlers

import (
	"net/http"

	"nomorewaste/internal/auth"
)

var manageableRoles = map[string]bool{
	"admin":     true,
	"volunteer": true,
	"merchant":  true,
	"member":    true,
}

var manageableStatuses = map[string]bool{
	"active": true,
	"banned": true,
}

type updateUserRequest struct {
	FullName string `json:"full_name"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

func (a *App) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateUserRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.FullName == "" || !manageableRoles[req.Role] || !manageableStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "invalid fields")
		return
	}
	identity, _ := auth.FromContext(r.Context())
	if identity != nil && identity.UserID == id && (req.Role != "admin" || req.Status != "active") {
		writeError(w, http.StatusForbidden, "cannot change your own admin access")
		return
	}
	result, err := a.DB.Exec("UPDATE users SET full_name = ?, role = ?, status = ? WHERE id = ?",
		req.FullName, req.Role, req.Status, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "user updated"})
}

func (a *App) SetUserStatus(w http.ResponseWriter, r *http.Request) {
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
	if !manageableStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	identity, _ := auth.FromContext(r.Context())
	if identity != nil && identity.UserID == id && req.Status == "banned" {
		writeError(w, http.StatusForbidden, "cannot ban your own account")
		return
	}
	if _, err := a.DB.Exec("UPDATE users SET status = ? WHERE id = ?", req.Status, id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "status updated"})
}

func (a *App) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	identity, _ := auth.FromContext(r.Context())
	if identity != nil && identity.UserID == id {
		writeError(w, http.StatusForbidden, "cannot delete your own account")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM users WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}

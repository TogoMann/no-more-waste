package handlers

import (
	"net/http"

	"nomorewaste/internal/models"
)

type volunteerRequest struct {
	FullName string  `json:"full_name"`
	Email    string  `json:"email"`
	Phone    string  `json:"phone"`
	SkillIDs []int64 `json:"skill_ids"`
}

func (a *App) ListSkills(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.Query("SELECT id, name FROM skills ORDER BY name")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	skills := []models.Skill{}
	for rows.Next() {
		var s models.Skill
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			continue
		}
		skills = append(skills, s)
	}
	writeJSON(w, http.StatusOK, skills)
}

func (a *App) volunteerSkills(volunteerID int64) []models.Skill {
	rows, err := a.DB.Query(`SELECT s.id, s.name FROM skills s
		JOIN volunteer_skills vs ON vs.skill_id = s.id WHERE vs.volunteer_id = ? ORDER BY s.name`, volunteerID)
	if err != nil {
		return []models.Skill{}
	}
	defer rows.Close()
	skills := []models.Skill{}
	for rows.Next() {
		var s models.Skill
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			continue
		}
		skills = append(skills, s)
	}
	return skills
}

func (a *App) ListVolunteers(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	query := `SELECT id, user_id, full_name, email, phone, status, created_at FROM volunteers`
	args := []interface{}{}
	if status != "" {
		query += " WHERE status = ?"
		args = append(args, status)
	}
	query += " ORDER BY id DESC"
	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	volunteers := []models.Volunteer{}
	for rows.Next() {
		var v models.Volunteer
		if err := rows.Scan(&v.ID, &v.UserID, &v.FullName, &v.Email, &v.Phone, &v.Status, &v.CreatedAt); err != nil {
			continue
		}
		v.Skills = a.volunteerSkills(v.ID)
		volunteers = append(volunteers, v)
	}
	writeJSON(w, http.StatusOK, volunteers)
}

func (a *App) CreateVolunteer(w http.ResponseWriter, r *http.Request) {
	var req volunteerRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.FullName == "" || req.Email == "" {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	res, err := tx.Exec(`INSERT INTO volunteers (full_name, email, phone, status) VALUES (?, ?, ?, 'pending')`,
		req.FullName, req.Email, req.Phone)
	if err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	volunteerID, _ := res.LastInsertId()
	for _, skillID := range req.SkillIDs {
		if _, err := tx.Exec(`INSERT INTO volunteer_skills (volunteer_id, skill_id) VALUES (?, ?)`,
			volunteerID, skillID); err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	v := models.Volunteer{ID: volunteerID, FullName: req.FullName, Email: req.Email,
		Phone: req.Phone, Status: "pending", Skills: a.volunteerSkills(volunteerID)}
	writeJSON(w, http.StatusCreated, v)
}

func (a *App) SetVolunteerStatus(w http.ResponseWriter, r *http.Request) {
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
	if req.Status != "approved" && req.Status != "rejected" && req.Status != "pending" {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	if _, err := a.DB.Exec("UPDATE volunteers SET status = ? WHERE id = ?", req.Status, id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "volunteer status updated"})
}

func (a *App) DeleteVolunteer(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM volunteers WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "volunteer deleted"})
}

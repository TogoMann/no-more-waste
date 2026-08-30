package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"nomorewaste/internal/auth"
	"nomorewaste/internal/exports"
	"nomorewaste/internal/models"
)

type planningSlotRequest struct {
	VolunteerID int64  `json:"volunteer_id"`
	Task        string `json:"task"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
}

type planningRequest struct {
	EventType       string                `json:"event_type"`
	PlanningDate    string                `json:"planning_date"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	Location        string                `json:"location"`
	StartTime       string                `json:"start_time"`
	EndTime         string                `json:"end_time"`
	MaxParticipants int                   `json:"max_participants"`
	Slots           []planningSlotRequest `json:"slots"`
}

func isPastDate(value string) bool {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return false
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return parsed.Before(today)
}

func (a *App) loadPlanning(id int64, viewerID int64) (models.Planning, error) {
	var p models.Planning
	err := a.DB.QueryRow(`SELECT p.id, p.planning_date, p.title, COALESCE(p.description, ''),
		COALESCE(p.location, ''), p.start_time, p.end_time, p.max_participants, p.event_type,
		p.created_by, COALESCE(u.full_name, ''), p.approval_status, COALESCE(p.review_note, ''), p.created_at,
		(SELECT COUNT(*) FROM planning_participants WHERE planning_id = p.id)
		FROM plannings p LEFT JOIN users u ON u.id = p.created_by WHERE p.id = ?`, id).
		Scan(&p.ID, &p.PlanningDate, &p.Title, &p.Description, &p.Location, &p.StartTime, &p.EndTime,
			&p.MaxParticipants, &p.EventType, &p.CreatedBy, &p.CreatorName, &p.ApprovalStatus,
			&p.ReviewNote, &p.CreatedAt, &p.ParticipantCount)
	if err != nil {
		return p, err
	}
	if viewerID > 0 {
		var exists int
		a.DB.QueryRow("SELECT COUNT(*) FROM planning_participants WHERE planning_id = ? AND user_id = ?",
			id, viewerID).Scan(&exists)
		p.Joined = exists > 0
	}
	rows, err := a.DB.Query(`SELECT ps.id, ps.planning_id, ps.volunteer_id, v.full_name, ps.task, ps.start_time, ps.end_time
		FROM planning_slots ps JOIN volunteers v ON v.id = ps.volunteer_id WHERE ps.planning_id = ?
		ORDER BY ps.start_time`, id)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	p.Slots = []models.PlanningSlot{}
	for rows.Next() {
		var slot models.PlanningSlot
		if err := rows.Scan(&slot.ID, &slot.PlanningID, &slot.VolunteerID, &slot.VolunteerName,
			&slot.Task, &slot.StartTime, &slot.EndTime); err != nil {
			continue
		}
		p.Slots = append(p.Slots, slot)
	}
	return p, nil
}

func (a *App) planningParticipants(planningID int64) []models.Participant {
	rows, err := a.DB.Query(`SELECT u.id, u.full_name, u.email, pp.joined_at
		FROM planning_participants pp JOIN users u ON u.id = pp.user_id
		WHERE pp.planning_id = ? ORDER BY pp.joined_at`, planningID)
	if err != nil {
		return []models.Participant{}
	}
	defer rows.Close()
	participants := []models.Participant{}
	for rows.Next() {
		var participant models.Participant
		if err := rows.Scan(&participant.UserID, &participant.FullName, &participant.Email,
			&participant.JoinedAt); err != nil {
			continue
		}
		participants = append(participants, participant)
	}
	return participants
}

func (a *App) ListPlannings(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.FromContext(r.Context())
	var viewerID int64
	role := ""
	if identity != nil {
		viewerID = identity.UserID
		role = identity.Role
	}
	query := `SELECT p.id, p.planning_date, p.title, COALESCE(p.description, ''),
		COALESCE(p.location, ''), p.start_time, p.end_time, p.max_participants, p.event_type,
		p.created_by, COALESCE(u.full_name, ''), p.approval_status, COALESCE(p.review_note, ''), p.created_at,
		(SELECT COUNT(*) FROM planning_participants WHERE planning_id = p.id),
		(SELECT COUNT(*) FROM planning_participants WHERE planning_id = p.id AND user_id = ?)
		FROM plannings p LEFT JOIN users u ON u.id = p.created_by`
	args := []interface{}{viewerID}
	if role != "admin" {
		query += " WHERE (p.approval_status = 'approved' OR p.created_by = ?)"
		args = append(args, viewerID)
	} else if status := r.URL.Query().Get("approval_status"); status != "" {
		query += " WHERE p.approval_status = ?"
		args = append(args, status)
	}
	query += " ORDER BY p.planning_date DESC"
	rows, err := a.DB.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	plannings := []models.Planning{}
	for rows.Next() {
		var p models.Planning
		var joined int
		if err := rows.Scan(&p.ID, &p.PlanningDate, &p.Title, &p.Description, &p.Location,
			&p.StartTime, &p.EndTime, &p.MaxParticipants, &p.EventType, &p.CreatedBy, &p.CreatorName,
			&p.ApprovalStatus, &p.ReviewNote, &p.CreatedAt, &p.ParticipantCount, &joined); err != nil {
			continue
		}
		p.Joined = joined > 0
		plannings = append(plannings, p)
	}
	writeJSON(w, http.StatusOK, plannings)
}

func (a *App) MyCreatedEvents(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := a.DB.Query(`SELECT p.id, p.planning_date, p.title, COALESCE(p.description, ''),
		COALESCE(p.location, ''), p.start_time, p.end_time, p.max_participants, p.event_type,
		p.created_by, COALESCE(u.full_name, ''), p.approval_status, COALESCE(p.review_note, ''), p.created_at,
		(SELECT COUNT(*) FROM planning_participants WHERE planning_id = p.id)
		FROM plannings p LEFT JOIN users u ON u.id = p.created_by
		WHERE p.created_by = ? ORDER BY p.planning_date DESC`, identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	plannings := []models.Planning{}
	for rows.Next() {
		var p models.Planning
		if err := rows.Scan(&p.ID, &p.PlanningDate, &p.Title, &p.Description, &p.Location,
			&p.StartTime, &p.EndTime, &p.MaxParticipants, &p.EventType, &p.CreatedBy, &p.CreatorName,
			&p.ApprovalStatus, &p.ReviewNote, &p.CreatedAt, &p.ParticipantCount); err != nil {
			continue
		}
		plannings = append(plannings, p)
	}
	writeJSON(w, http.StatusOK, plannings)
}

func (a *App) ReviewPlanning(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	identity, _ := auth.FromContext(r.Context())
	var req struct {
		Status     string `json:"status"`
		ReviewNote string `json:"review_note"`
	}
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Status != "approved" && req.Status != "rejected" {
		writeError(w, http.StatusBadRequest, "statut invalide")
		return
	}
	var reviewerID interface{}
	if identity != nil {
		reviewerID = identity.UserID
	}
	result, err := a.DB.Exec(`UPDATE plannings SET approval_status = ?, review_note = ?, reviewed_by = ?
		WHERE id = ?`, req.Status, req.ReviewNote, reviewerID, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeError(w, http.StatusNotFound, "evenement introuvable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "evenement " + req.Status})
}

func (a *App) GetPlanning(w http.ResponseWriter, r *http.Request) {
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
	planning, err := a.loadPlanning(id, viewerID)
	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "planning not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if identity != nil && identity.Role == "admin" {
		planning.Participants = a.planningParticipants(id)
	}
	writeJSON(w, http.StatusOK, planning)
}

func (a *App) CreatePlanning(w http.ResponseWriter, r *http.Request) {
	var req planningRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.PlanningDate == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	if isPastDate(req.PlanningDate) {
		writeError(w, http.StatusBadRequest, "cannot create an event in the past")
		return
	}
	if req.StartTime == "" {
		req.StartTime = "09:00"
	}
	if req.EndTime == "" {
		req.EndTime = "17:00"
	}
	if req.MaxParticipants <= 0 {
		req.MaxParticipants = 10
	}
	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	approvalStatus := "approved"
	var creatorID interface{}
	if identity, ok := auth.FromContext(r.Context()); ok {
		creatorID = identity.UserID
		if identity.Role != "admin" {
			approvalStatus = "pending"
		}
	}
	eventType := req.EventType
	if eventType == "" {
		eventType = "mission"
	}
	res, err := tx.Exec(`INSERT INTO plannings (planning_date, title, description, location, start_time,
		end_time, max_participants, event_type, created_by, approval_status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.PlanningDate, req.Title, req.Description, req.Location, req.StartTime, req.EndTime,
		req.MaxParticipants, eventType, creatorID, approvalStatus)
	if err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	planningID, _ := res.LastInsertId()
	for _, slot := range req.Slots {
		if _, err := tx.Exec(`INSERT INTO planning_slots (planning_id, volunteer_id, task, start_time, end_time)
			VALUES (?, ?, ?, ?, ?)`, planningID, slot.VolunteerID, slot.Task, slot.StartTime, slot.EndTime); err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	planning, _ := a.loadPlanning(planningID, 0)
	writeJSON(w, http.StatusCreated, planning)
}

func (a *App) UpdatePlanning(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req planningRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.PlanningDate == "" || req.Title == "" {
		writeError(w, http.StatusBadRequest, "missing fields")
		return
	}
	var currentDate string
	if err := a.DB.QueryRow("SELECT planning_date FROM plannings WHERE id = ?", id).Scan(&currentDate); err != nil {
		writeError(w, http.StatusNotFound, "planning not found")
		return
	}
	if isPastDate(currentDate) {
		writeError(w, http.StatusBadRequest, "cannot modify a past event")
		return
	}
	if isPastDate(req.PlanningDate) {
		writeError(w, http.StatusBadRequest, "cannot move an event to the past")
		return
	}
	if req.StartTime == "" {
		req.StartTime = "09:00"
	}
	if req.EndTime == "" {
		req.EndTime = "17:00"
	}
	if req.MaxParticipants <= 0 {
		req.MaxParticipants = 10
	}
	var participantCount int
	a.DB.QueryRow("SELECT COUNT(*) FROM planning_participants WHERE planning_id = ?", id).Scan(&participantCount)
	if req.MaxParticipants < participantCount {
		writeError(w, http.StatusBadRequest, "capacity below current participants")
		return
	}
	tx, err := a.DB.Begin()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec(`UPDATE plannings SET planning_date = ?, title = ?, description = ?, location = ?,
		start_time = ?, end_time = ?, max_participants = ?,
		event_type = COALESCE(NULLIF(?, ''), event_type) WHERE id = ?`,
		req.PlanningDate, req.Title, req.Description, req.Location, req.StartTime, req.EndTime,
		req.MaxParticipants, req.EventType, id); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	if _, err := tx.Exec("DELETE FROM planning_slots WHERE planning_id = ?", id); err != nil {
		tx.Rollback()
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	for _, slot := range req.Slots {
		if _, err := tx.Exec(`INSERT INTO planning_slots (planning_id, volunteer_id, task, start_time, end_time)
			VALUES (?, ?, ?, ?, ?)`, id, slot.VolunteerID, slot.Task, slot.StartTime, slot.EndTime); err != nil {
			tx.Rollback()
			writeError(w, http.StatusInternalServerError, "database error")
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	planning, _ := a.loadPlanning(id, 0)
	writeJSON(w, http.StatusOK, planning)
}

func (a *App) DeletePlanning(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var currentDate string
	if err := a.DB.QueryRow("SELECT planning_date FROM plannings WHERE id = ?", id).Scan(&currentDate); err != nil {
		writeError(w, http.StatusNotFound, "planning not found")
		return
	}
	if isPastDate(currentDate) {
		writeError(w, http.StatusBadRequest, "cannot delete a past event")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM plannings WHERE id = ?", id); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "planning deleted"})
}

func (a *App) JoinPlanning(w http.ResponseWriter, r *http.Request) {
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
	var planningDate, approvalStatus string
	var maxParticipants int
	err = a.DB.QueryRow("SELECT planning_date, max_participants, approval_status FROM plannings WHERE id = ?", id).
		Scan(&planningDate, &maxParticipants, &approvalStatus)
	if err != nil {
		writeError(w, http.StatusNotFound, "planning not found")
		return
	}
	if approvalStatus != "approved" {
		writeError(w, http.StatusBadRequest, "cet evenement n est pas encore valide")
		return
	}
	if isPastDate(planningDate) {
		writeError(w, http.StatusBadRequest, "event already passed")
		return
	}
	var count int
	a.DB.QueryRow("SELECT COUNT(*) FROM planning_participants WHERE planning_id = ?", id).Scan(&count)
	if count >= maxParticipants {
		writeError(w, http.StatusConflict, "event is full")
		return
	}
	_, err = a.DB.Exec("INSERT OR IGNORE INTO planning_participants (planning_id, user_id) VALUES (?, ?)",
		id, identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "joined"})
}

func (a *App) LeavePlanning(w http.ResponseWriter, r *http.Request) {
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
	var planningDate string
	if err := a.DB.QueryRow("SELECT planning_date FROM plannings WHERE id = ?", id).Scan(&planningDate); err != nil {
		writeError(w, http.StatusNotFound, "planning not found")
		return
	}
	if isPastDate(planningDate) {
		writeError(w, http.StatusBadRequest, "event already passed")
		return
	}
	if _, err := a.DB.Exec("DELETE FROM planning_participants WHERE planning_id = ? AND user_id = ?",
		id, identity.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "left"})
}

func (a *App) MyPlannings(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	rows, err := a.DB.Query(`SELECT p.id, p.planning_date, p.title, COALESCE(p.description, ''),
		COALESCE(p.location, ''), p.start_time, p.end_time, p.max_participants, p.created_at,
		(SELECT COUNT(*) FROM planning_participants WHERE planning_id = p.id)
		FROM plannings p JOIN planning_participants pp ON pp.planning_id = p.id
		WHERE pp.user_id = ? ORDER BY p.planning_date DESC`, identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	defer rows.Close()
	plannings := []models.Planning{}
	for rows.Next() {
		var p models.Planning
		if err := rows.Scan(&p.ID, &p.PlanningDate, &p.Title, &p.Description, &p.Location,
			&p.StartTime, &p.EndTime, &p.MaxParticipants, &p.CreatedAt, &p.ParticipantCount); err != nil {
			continue
		}
		p.Joined = true
		plannings = append(plannings, p)
	}
	writeJSON(w, http.StatusOK, plannings)
}

func (a *App) PlanningExcel(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	planning, err := a.loadPlanning(id, 0)
	if err != nil {
		writeError(w, http.StatusNotFound, "planning not found")
		return
	}
	data, err := exports.PlanningExcel(planning)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "excel error")
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=planning-%d.xlsx", id))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

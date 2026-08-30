package handlers

import (
	"database/sql"
	"io"
	"net/http"
	"strconv"
	"strings"

	"nomorewaste/internal/auth"
	"nomorewaste/internal/models"
	"nomorewaste/internal/payments"
)

type profileRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Address  string `json:"address"`
	CityID   *int64 `json:"city_id"`
}

type passwordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (a *App) GetProfile(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := a.loadUserProfile(identity.UserID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (a *App) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req profileRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	_, err := a.DB.Exec("UPDATE users SET full_name = ?, phone = ?, address = ?, city_id = ? WHERE id = ?",
		req.FullName, req.Phone, req.Address, req.CityID, identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "profile updated"})
}

func (a *App) ChangePassword(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req passwordRequest
	if err := decodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	if len(req.NewPassword) < 6 {
		writeError(w, http.StatusBadRequest, "password too short")
		return
	}
	var currentHash string
	if err := a.DB.QueryRow("SELECT password_hash FROM users WHERE id = ?", identity.UserID).Scan(&currentHash); err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	if !auth.CheckPassword(req.CurrentPassword, currentHash) {
		writeError(w, http.StatusUnauthorized, "wrong current password")
		return
	}
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash error")
		return
	}
	if _, err := a.DB.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHash, identity.UserID); err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

func (a *App) loadUserProfile(userID int64) (models.User, error) {
	var user models.User
	var hasPaid int
	var membershipEnd sql.NullString
	err := a.DB.QueryRow(`SELECT u.id, u.email, u.full_name, u.role, u.status, COALESCE(u.phone, ''),
		COALESCE(u.address, ''), u.city_id, COALESCE(c.name, ''), u.membership_end_date,
		u.has_paid_dues, u.created_at
		FROM users u LEFT JOIN cities c ON c.id = u.city_id WHERE u.id = ?`, userID).
		Scan(&user.ID, &user.Email, &user.FullName, &user.Role, &user.Status, &user.Phone,
			&user.Address, &user.CityID, &user.CityName, &membershipEnd, &hasPaid, &user.CreatedAt)
	if err != nil {
		return user, err
	}
	user.MembershipEndDate = membershipEnd.String
	user.HasPaidDues = hasPaid > 0
	user.MembershipValid = a.membershipIsValid(userID)
	return user, nil
}

func (a *App) grantMembershipYear(userID int64) (string, error) {
	newEnd := truncateToday().AddDate(1, 0, 0).Format("2006-01-02")
	_, err := a.DB.Exec("UPDATE users SET has_paid_dues = 1, membership_end_date = ? WHERE id = ?",
		newEnd, userID)
	return newEnd, err
}

func (a *App) DuesInfo(w http.ResponseWriter, r *http.Request) {
	publicKey := ""
	if a.Stripe != nil {
		publicKey = a.Stripe.PublicKey
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"amount_cents":   a.DuesCents,
		"currency":       "eur",
		"stripe_enabled": a.Stripe != nil && a.Stripe.Enabled(),
		"public_key":     publicKey,
	})
}

func (a *App) CreateDuesCheckout(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if a.Stripe == nil || !a.Stripe.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "paiement indisponible")
		return
	}
	session, err := a.Stripe.CreateCheckoutSession(payments.CheckoutRequest{
		AmountCents:   a.DuesCents,
		ProductName:   "Cotisation annuelle NO MORE WASTE",
		Description:   "Adhesion valable 1 an a compter du paiement",
		CustomerEmail: identity.Email,
		ClientRef:     strconv.FormatInt(identity.UserID, 10),
		SuccessURL:    a.PublicURL + "/espace/profil?session_id={CHECKOUT_SESSION_ID}",
		CancelURL:     a.PublicURL + "/espace/profil?payment=cancelled",
	})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	a.DB.Exec(`INSERT OR IGNORE INTO payments (user_id, session_id, amount_cents, currency, status)
		VALUES (?, ?, ?, 'eur', 'pending')`, identity.UserID, session.ID, a.DuesCents)
	writeJSON(w, http.StatusOK, map[string]string{"session_id": session.ID, "url": session.URL})
}

func (a *App) ConfirmDuesPayment(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := decodeBody(r, &req); err != nil || req.SessionID == "" {
		writeError(w, http.StatusBadRequest, "session_id manquant")
		return
	}
	if a.Stripe == nil || !a.Stripe.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "paiement indisponible")
		return
	}
	session, err := a.Stripe.RetrieveCheckoutSession(req.SessionID)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	if session.ClientReference != strconv.FormatInt(identity.UserID, 10) {
		writeError(w, http.StatusForbidden, "cette session ne vous appartient pas")
		return
	}
	if session.PaymentStatus != "paid" {
		writeError(w, http.StatusPaymentRequired, "paiement non finalise")
		return
	}
	newEnd, err := a.grantMembershipYear(identity.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "database error")
		return
	}
	a.DB.Exec(`UPDATE payments SET status = 'paid', paid_at = datetime('now'), payment_intent_id = ?
		WHERE session_id = ?`, session.PaymentIntent, session.ID)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":             "cotisation reglee",
		"membership_end_date": newEnd,
		"membership_valid":    true,
	})
}

func (a *App) StripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "corps illisible")
		return
	}
	defer r.Body.Close()
	if a.Stripe == nil {
		writeError(w, http.StatusServiceUnavailable, "paiement indisponible")
		return
	}
	event, err := a.Stripe.VerifyWebhook(payload, r.Header.Get("Stripe-Signature"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if event.Type == "checkout.session.completed" && event.Data.Object.PaymentStatus == "paid" {
		userID, convErr := strconv.ParseInt(event.Data.Object.ClientReference, 10, 64)
		if convErr == nil && userID > 0 {
			a.grantMembershipYear(userID)
			a.DB.Exec(`UPDATE payments SET status = 'paid', paid_at = datetime('now') WHERE session_id = ?`,
				event.Data.Object.ID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"received": "true"})
}

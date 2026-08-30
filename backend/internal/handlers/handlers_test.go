package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"nomorewaste/internal/auth"
	"nomorewaste/internal/database"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.RunScript(db, "../../../database/schema.sql"); err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := database.RunScript(db, "../../../database/seed.sql"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return &App{DB: db, JWTSecret: "test-secret"}
}

func doJSON(t *testing.T, handler http.HandlerFunc, method, target string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		payload, _ := json.Marshal(body)
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, target, reader)
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

func TestRegisterAndLogin(t *testing.T) {
	app := newTestApp(t)

	registerBody := map[string]string{"email": "user@test.fr", "password": "secret123", "full_name": "Test User", "role": "member"}
	response := doJSON(t, app.Register, http.MethodPost, "/api/auth/register", registerBody)
	if response.Code != http.StatusCreated {
		t.Fatalf("register status = %d, body = %s", response.Code, response.Body.String())
	}
	var registered authResponse
	json.Unmarshal(response.Body.Bytes(), &registered)
	if registered.Token == "" {
		t.Fatal("expected token after register")
	}

	loginBody := map[string]string{"email": "user@test.fr", "password": "secret123"}
	loginResponse := doJSON(t, app.Login, http.MethodPost, "/api/auth/login", loginBody)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status = %d", loginResponse.Code)
	}

	wrongLogin := doJSON(t, app.Login, http.MethodPost, "/api/auth/login", map[string]string{"email": "user@test.fr", "password": "bad"})
	if wrongLogin.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for bad password, got %d", wrongLogin.Code)
	}
}

func TestMerchantCreateAndList(t *testing.T) {
	app := newTestApp(t)

	body := map[string]string{"company_name": "Epicerie", "contact_name": "Alice", "email": "a@test.fr"}
	response := doJSON(t, app.CreateMerchant, http.MethodPost, "/api/merchants", body)
	if response.Code != http.StatusCreated {
		t.Fatalf("create merchant status = %d, body = %s", response.Code, response.Body.String())
	}

	listResponse := doJSON(t, app.ListMerchants, http.MethodGet, "/api/merchants", nil)
	var merchants []map[string]interface{}
	json.Unmarshal(listResponse.Body.Bytes(), &merchants)
	if len(merchants) != 1 {
		t.Fatalf("expected 1 merchant, got %d", len(merchants))
	}
}

func TestProductStockMovement(t *testing.T) {
	app := newTestApp(t)

	createResponse := doJSON(t, app.CreateProduct, http.MethodPost, "/api/products", map[string]interface{}{
		"name": "Pain", "quantity": 10, "expiration_date": futureDate(5)})
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create product status = %d", createResponse.Code)
	}

	request := httptest.NewRequest(http.MethodPost, "/api/products/1/stock", bytes.NewReader(mustJSON(map[string]interface{}{"movement_type": "out", "quantity": 4})))
	request.SetPathValue("id", "1")
	recorder := httptest.NewRecorder()
	app.MoveStock(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("move stock status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &result)
	if result["quantity"].(float64) != 6 {
		t.Fatalf("expected quantity 6, got %v", result["quantity"])
	}
}

func TestVolunteerLifecycle(t *testing.T) {
	app := newTestApp(t)

	createResponse := doJSON(t, app.CreateVolunteer, http.MethodPost, "/api/volunteers", map[string]interface{}{
		"full_name": "Marie", "email": "marie@test.fr", "skill_ids": []int64{1, 2},
	})
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create volunteer status = %d", createResponse.Code)
	}

	request := httptest.NewRequest(http.MethodPatch, "/api/volunteers/1/status", bytes.NewReader(mustJSON(map[string]string{"status": "approved"})))
	request.SetPathValue("id", "1")
	recorder := httptest.NewRecorder()
	app.SetVolunteerStatus(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("approve volunteer status = %d", recorder.Code)
	}
}

func createAdmin(t *testing.T, app *App) int64 {
	t.Helper()
	hash, _ := auth.HashPassword("admin123")
	result, err := app.DB.Exec(`INSERT INTO users (email, password_hash, full_name, role)
		VALUES ('reviewer@test.fr', ?, 'Reviewer', 'admin')`, hash)
	if err != nil {
		t.Fatalf("creation admin: %v", err)
	}
	id, _ := result.LastInsertId()
	return id
}

func futureDate(days int) string {
	return time.Now().AddDate(0, 0, days).Format("2006-01-02")
}

func TestProductRequiresExpirationDate(t *testing.T) {
	app := newTestApp(t)
	response := doJSON(t, app.CreateProduct, http.MethodPost, "/api/products",
		map[string]interface{}{"name": "Sans DLC", "quantity": 5})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without expiration date, got %d", response.Code)
	}
}

func TestServiceCategoryValidation(t *testing.T) {
	app := newTestApp(t)
	bad := doJSON(t, app.CreateService, http.MethodPost, "/api/services", map[string]interface{}{
		"title": "Invalide", "category": "yoga", "date_time": futureDate(3) + " 10:00"})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid category, got %d", bad.Code)
	}
	good := doJSON(t, app.CreateService, http.MethodPost, "/api/services", map[string]interface{}{
		"title": "Atelier cuisine", "category": "cuisine", "date_time": futureDate(3) + " 10:00",
		"max_capacity": 5})
	if good.Code != http.StatusCreated {
		t.Fatalf("expected 201 for valid service, got %d body %s", good.Code, good.Body.String())
	}
}

func TestServiceSubscriptionRequiresDues(t *testing.T) {
	app := newTestApp(t)
	doJSON(t, app.CreateService, http.MethodPost, "/api/services", map[string]interface{}{
		"title": "Atelier", "category": "bricolage", "date_time": futureDate(4) + " 14:00", "max_capacity": 2})

	hash, _ := auth.HashPassword("secret123")
	result, err := app.DB.Exec(`INSERT INTO users (email, password_hash, full_name, role, has_paid_dues)
		VALUES ('unpaid@test.fr', ?, 'Unpaid', 'member', 0)`, hash)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	userID, _ := result.LastInsertId()

	request := httptest.NewRequest(http.MethodPost, "/api/services/1/subscribe", nil)
	request.SetPathValue("id", "1")
	request = request.WithContext(auth.WithIdentity(request.Context(),
		&auth.Identity{UserID: userID, Email: "unpaid@test.fr", Role: "member"}))
	recorder := httptest.NewRecorder()
	app.SubscribeService(recorder, request)
	if recorder.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402 without dues, got %d", recorder.Code)
	}

	if _, err := app.grantMembershipYear(userID); err != nil {
		t.Fatalf("grant membership failed: %v", err)
	}

	retryRequest := httptest.NewRequest(http.MethodPost, "/api/services/1/subscribe", nil)
	retryRequest.SetPathValue("id", "1")
	retryRequest = retryRequest.WithContext(auth.WithIdentity(retryRequest.Context(),
		&auth.Identity{UserID: userID, Email: "unpaid@test.fr", Role: "member"}))
	retryRecorder := httptest.NewRecorder()
	app.SubscribeService(retryRecorder, retryRequest)
	if retryRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200 after paying dues, got %d body %s", retryRecorder.Code, retryRecorder.Body.String())
	}
}

func TestCompleteCollectionStoresProducts(t *testing.T) {
	app := newTestApp(t)
	app.DB.Exec(`INSERT INTO merchants (company_name, contact_name, email, membership_end)
		VALUES ('Test', 'Contact', 'a@b.fr', '2030-01-01')`)
	createResponse := doJSON(t, app.CreateCollection, http.MethodPost, "/api/collections", map[string]interface{}{
		"label": "Ramassage", "scheduled_date": futureDate(1),
		"stops": []map[string]interface{}{{"merchant_id": 1, "order_index": 1}}})
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create collection failed: %d %s", createResponse.Code, createResponse.Body.String())
	}

	missingDLC := httptest.NewRequest(http.MethodPatch, "/api/collections/1/complete",
		bytes.NewReader(mustJSON(map[string]interface{}{
			"products": []map[string]interface{}{{"name": "Pain", "quantity": 10}}})))
	missingDLC.SetPathValue("id", "1")
	missingRecorder := httptest.NewRecorder()
	app.CompleteCollection(missingRecorder, missingDLC)
	if missingRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without expiration date, got %d", missingRecorder.Code)
	}

	request := httptest.NewRequest(http.MethodPatch, "/api/collections/1/complete",
		bytes.NewReader(mustJSON(map[string]interface{}{
			"products": []map[string]interface{}{
				{"name": "Pain", "quantity": 10, "expiration_date": futureDate(2)},
				{"name": "Pommes", "quantity": 5, "expiration_date": futureDate(6)}}})))
	request.SetPathValue("id", "1")
	recorder := httptest.NewRecorder()
	app.CompleteCollection(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("complete failed: %d %s", recorder.Code, recorder.Body.String())
	}

	var productCount int
	app.DB.QueryRow("SELECT COUNT(*) FROM products").Scan(&productCount)
	if productCount != 2 {
		t.Fatalf("expected 2 products stored, got %d", productCount)
	}
	var status string
	app.DB.QueryRow("SELECT status FROM collections WHERE id = 1").Scan(&status)
	if status != "completed" {
		t.Fatalf("expected completed status, got %s", status)
	}
}

func TestCollectedProductsAppearInListing(t *testing.T) {
	app := newTestApp(t)
	app.DB.Exec(`INSERT INTO merchants (company_name, contact_name, email, membership_end)
		VALUES ('Test', 'Contact', 'a@b.fr', '2030-01-01')`)
	doJSON(t, app.CreateCollection, http.MethodPost, "/api/collections", map[string]interface{}{
		"label": "Ramassage", "scheduled_date": futureDate(1),
		"stops": []map[string]interface{}{{"merchant_id": 1, "order_index": 1}}})

	request := httptest.NewRequest(http.MethodPatch, "/api/collections/1/complete",
		bytes.NewReader(mustJSON(map[string]interface{}{
			"products": []map[string]interface{}{
				{"name": "Pain collecte", "barcode": "COLLECT1", "quantity": 12,
					"expiration_date": futureDate(2)}}})))
	request.SetPathValue("id", "1")
	recorder := httptest.NewRecorder()
	app.CompleteCollection(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("complete failed: %d", recorder.Code)
	}

	listResponse := doJSON(t, app.ListProducts, http.MethodGet, "/api/products", nil)
	var products []map[string]interface{}
	json.Unmarshal(listResponse.Body.Bytes(), &products)
	if len(products) != 1 {
		t.Fatalf("expected collected product in listing, got %d", len(products))
	}

	barcodeRequest := httptest.NewRequest(http.MethodGet, "/api/barcodes/COLLECT1", nil)
	barcodeRequest.SetPathValue("barcode", "COLLECT1")
	barcodeRecorder := httptest.NewRecorder()
	app.GetProductByBarcode(barcodeRecorder, barcodeRequest)
	if barcodeRecorder.Code != http.StatusOK {
		t.Fatalf("expected barcode lookup to succeed, got %d", barcodeRecorder.Code)
	}
}

func TestDashboardExpiringProducts(t *testing.T) {
	app := newTestApp(t)
	app.DB.Exec(`INSERT INTO products (name, barcode, quantity, expiration_date)
		VALUES ('Critique', 'B1', 5, ?)`, futureDate(1))
	app.DB.Exec(`INSERT INTO products (name, barcode, quantity, expiration_date)
		VALUES ('Sain', 'B2', 5, ?)`, futureDate(30))

	response := doJSON(t, app.Dashboard, http.MethodGet, "/api/dashboard", nil)
	var stats map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &stats)
	if stats["expiring_count"].(float64) != 1 {
		t.Fatalf("expected 1 expiring product, got %v", stats["expiring_count"])
	}
	products := stats["expiring_products"].([]interface{})
	if len(products) != 1 {
		t.Fatalf("expected 1 product in alert list, got %d", len(products))
	}
}

func mustJSON(value interface{}) []byte {
	payload, _ := json.Marshal(value)
	return payload
}

func TestMembershipLastsExactlyOneYear(t *testing.T) {
	app := newTestApp(t)
	hash, _ := auth.HashPassword("secret123")
	result, _ := app.DB.Exec(`INSERT INTO users (email, password_hash, full_name, role, has_paid_dues,
		membership_end_date) VALUES ('year@test.fr', ?, 'Year', 'member', 1, ?)`, hash, futureDate(200))
	userID, _ := result.LastInsertId()

	newEnd, err := app.grantMembershipYear(userID)
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	expected := time.Now().AddDate(1, 0, 0).Format("2006-01-02")
	if newEnd != expected {
		t.Fatalf("la cotisation doit durer 1 an: attendu %s, obtenu %s", expected, newEnd)
	}
}

func TestVolunteerEventNeedsApproval(t *testing.T) {
	app := newTestApp(t)
	hash, _ := auth.HashPassword("secret123")
	result, _ := app.DB.Exec(`INSERT INTO users (email, password_hash, full_name, role)
		VALUES ('vol@test.fr', ?, 'Vol', 'volunteer')`, hash)
	volunteerID, _ := result.LastInsertId()

	request := httptest.NewRequest(http.MethodPost, "/api/events", bytes.NewReader(mustJSON(
		map[string]interface{}{"title": "Brocante", "planning_date": futureDate(10),
			"event_type": "brocante", "max_participants": 20})))
	request = request.WithContext(auth.WithIdentity(request.Context(),
		&auth.Identity{UserID: volunteerID, Email: "vol@test.fr", Role: "volunteer"}))
	recorder := httptest.NewRecorder()
	app.CreatePlanning(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("creation echouee: %d %s", recorder.Code, recorder.Body.String())
	}
	var created map[string]interface{}
	json.Unmarshal(recorder.Body.Bytes(), &created)
	if created["approval_status"] != "pending" {
		t.Fatalf("un evenement benevole doit etre en attente, obtenu %v", created["approval_status"])
	}

	joinRequest := httptest.NewRequest(http.MethodPost, "/api/plannings/1/join", nil)
	joinRequest.SetPathValue("id", "1")
	joinRequest = joinRequest.WithContext(auth.WithIdentity(joinRequest.Context(),
		&auth.Identity{UserID: volunteerID, Email: "vol@test.fr", Role: "volunteer"}))
	joinRecorder := httptest.NewRecorder()
	app.JoinPlanning(joinRecorder, joinRequest)
	if joinRecorder.Code != http.StatusBadRequest {
		t.Fatalf("on ne doit pas pouvoir rejoindre un evenement non valide: %d", joinRecorder.Code)
	}

	adminID := createAdmin(t, app)
	reviewRequest := httptest.NewRequest(http.MethodPatch, "/api/plannings/1/review",
		bytes.NewReader(mustJSON(map[string]string{"status": "approved"})))
	reviewRequest.SetPathValue("id", "1")
	reviewRequest = reviewRequest.WithContext(auth.WithIdentity(reviewRequest.Context(),
		&auth.Identity{UserID: adminID, Email: "reviewer@test.fr", Role: "admin"}))
	reviewRecorder := httptest.NewRecorder()
	app.ReviewPlanning(reviewRecorder, reviewRequest)
	if reviewRecorder.Code != http.StatusOK {
		t.Fatalf("validation echouee: %d", reviewRecorder.Code)
	}

	retryRecorder := httptest.NewRecorder()
	retryRequest := httptest.NewRequest(http.MethodPost, "/api/plannings/1/join", nil)
	retryRequest.SetPathValue("id", "1")
	retryRequest = retryRequest.WithContext(auth.WithIdentity(retryRequest.Context(),
		&auth.Identity{UserID: volunteerID, Email: "vol@test.fr", Role: "volunteer"}))
	app.JoinPlanning(retryRecorder, retryRequest)
	if retryRecorder.Code != http.StatusOK {
		t.Fatalf("inscription apres validation echouee: %d %s", retryRecorder.Code, retryRecorder.Body.String())
	}
}

func TestDonationApprovalCreatesCollection(t *testing.T) {
	app := newTestApp(t)
	hash, _ := auth.HashPassword("secret123")
	userResult, _ := app.DB.Exec(`INSERT INTO users (email, password_hash, full_name, role, company_name, siret)
		VALUES ('shop@test.fr', ?, 'Shop', 'merchant', 'MA BOUTIQUE', '35247171800010')`, hash)
	merchantUserID, _ := userResult.LastInsertId()
	app.DB.Exec(`INSERT INTO merchants (user_id, company_name, contact_name, email, membership_end)
		VALUES (?, 'MA BOUTIQUE', 'Shop', 'shop@test.fr', '2030-01-01')`, merchantUserID)

	createRequest := httptest.NewRequest(http.MethodPost, "/api/donations", bytes.NewReader(mustJSON(
		map[string]interface{}{"title": "Invendus du jour", "donation_type": "food",
			"quantity": 20, "expiration_date": futureDate(3)})))
	createRequest = createRequest.WithContext(auth.WithIdentity(createRequest.Context(),
		&auth.Identity{UserID: merchantUserID, Email: "shop@test.fr", Role: "merchant"}))
	createRecorder := httptest.NewRecorder()
	app.CreateDonation(createRecorder, createRequest)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("creation don echouee: %d %s", createRecorder.Code, createRecorder.Body.String())
	}

	missingDLC := httptest.NewRequest(http.MethodPost, "/api/donations", bytes.NewReader(mustJSON(
		map[string]interface{}{"title": "Sans DLC", "donation_type": "food", "quantity": 5})))
	missingDLC = missingDLC.WithContext(auth.WithIdentity(missingDLC.Context(),
		&auth.Identity{UserID: merchantUserID, Email: "shop@test.fr", Role: "merchant"}))
	missingRecorder := httptest.NewRecorder()
	app.CreateDonation(missingRecorder, missingDLC)
	if missingRecorder.Code != http.StatusBadRequest {
		t.Fatalf("un don alimentaire sans DLC doit etre refuse: %d", missingRecorder.Code)
	}

	adminID := createAdmin(t, app)
	reviewRequest := httptest.NewRequest(http.MethodPatch, "/api/donations/1/review",
		bytes.NewReader(mustJSON(map[string]interface{}{"status": "approved", "scheduled_date": futureDate(2)})))
	reviewRequest.SetPathValue("id", "1")
	reviewRequest = reviewRequest.WithContext(auth.WithIdentity(reviewRequest.Context(),
		&auth.Identity{UserID: adminID, Email: "reviewer@test.fr", Role: "admin"}))
	reviewRecorder := httptest.NewRecorder()
	app.ReviewDonation(reviewRecorder, reviewRequest)
	if reviewRecorder.Code != http.StatusOK {
		t.Fatalf("validation don echouee: %d %s", reviewRecorder.Code, reviewRecorder.Body.String())
	}

	var collectionCount, stopCount int
	app.DB.QueryRow("SELECT COUNT(*) FROM collections").Scan(&collectionCount)
	app.DB.QueryRow("SELECT COUNT(*) FROM collection_stops").Scan(&stopCount)
	if collectionCount != 1 || stopCount != 1 {
		t.Fatalf("la validation doit creer une collecte avec un arret: %d collectes, %d arrets",
			collectionCount, stopCount)
	}
	var status string
	app.DB.QueryRow("SELECT status FROM donation_offers WHERE id = 1").Scan(&status)
	if status != "scheduled" {
		t.Fatalf("le don doit passer en scheduled, obtenu %s", status)
	}

	completeRequest := httptest.NewRequest(http.MethodPatch, "/api/collections/1/complete",
		bytes.NewReader(mustJSON(map[string]interface{}{"products": []map[string]interface{}{
			{"name": "Invendus", "quantity": 20, "expiration_date": futureDate(3)}}})))
	completeRequest.SetPathValue("id", "1")
	completeRecorder := httptest.NewRecorder()
	app.CompleteCollection(completeRecorder, completeRequest)
	if completeRecorder.Code != http.StatusOK {
		t.Fatalf("cloture collecte echouee: %d", completeRecorder.Code)
	}
	app.DB.QueryRow("SELECT status FROM donation_offers WHERE id = 1").Scan(&status)
	if status != "collected" {
		t.Fatalf("le don doit passer en collected, obtenu %s", status)
	}
}

package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"nomorewaste/internal/auth"
	"nomorewaste/internal/database"
	"nomorewaste/internal/handlers"
	"nomorewaste/internal/payments"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func bootstrapAdmin(db *sql.DB) {
	email := env("ADMIN_EMAIL", "admin@nomorewaste.org")
	password := env("ADMIN_PASSWORD", "admin123")
	var count int
	db.QueryRow("SELECT COUNT(*) FROM users WHERE role = 'admin'").Scan(&count)
	if count > 0 {
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		log.Println("admin bootstrap failed:", err)
		return
	}
	_, err = db.Exec("INSERT INTO users (email, password_hash, full_name, role) VALUES (?, ?, ?, 'admin')",
		email, hash, "Administrateur")
	if err != nil {
		log.Println("admin bootstrap failed:", err)
		return
	}
	log.Printf("admin account created: %s", email)
}

func startRenewalReminderJob(db *sql.DB) {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			checkExpiredMemberships(db)
			<-ticker.C
		}
	}()
}

func checkExpiredMemberships(db *sql.DB) {
	rows, err := db.Query(`SELECT company_name, membership_end FROM merchants
		WHERE status = 'active' AND date(membership_end) <= date('now', '+30 day')`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var company, end string
		if err := rows.Scan(&company, &end); err == nil {
			log.Printf("[RAPPEL ADHESION] %s expire le %s", company, end)
		}
	}
}

func main() {
	dbPath := env("DB_PATH", "nomorewaste.db")
	schemaPath := env("SCHEMA_PATH", "../database/schema.sql")
	seedPath := env("SEED_PATH", "../database/seed.sql")
	jwtSecret := env("JWT_SECRET", "no-more-waste-secret-key")
	port := env("PORT", "8080")

	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatal("database open failed:", err)
	}
	defer db.Close()

	if err := database.RunScript(db, schemaPath); err != nil {
		log.Fatal("schema init failed:", err)
	}
	if err := database.RunScript(db, seedPath); err != nil {
		log.Println("seed skipped:", err)
	}

	bootstrapAdmin(db)
	if env("SEED_DEMO", "false") == "true" {
		seedDemoData(db)
	}
	startRenewalReminderJob(db)

	stripeClient := payments.NewClient(
		env("STRIPE_SECRET_KEY", ""),
		env("STRIPE_PUBLIC_KEY", ""),
		env("STRIPE_WEBHOOK_SECRET", ""),
	)
	duesCents := int64(2000)
	if raw := env("DUES_AMOUNT_CENTS", ""); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			duesCents = parsed
		}
	}
	if stripeClient.Enabled() {
		log.Println("stripe enabled")
	} else {
		log.Println("stripe disabled: STRIPE_SECRET_KEY missing")
	}

	app := &handlers.App{
		DB:        db,
		JWTSecret: jwtSecret,
		Stripe:    stripeClient,
		PublicURL: env("PUBLIC_URL", "http://localhost:9080"),
		DuesCents: duesCents,
	}
	mux := http.NewServeMux()

	protected := func(h http.HandlerFunc) http.HandlerFunc {
		return auth.Middleware(jwtSecret, h)
	}
	adminOnly := func(h http.HandlerFunc) http.HandlerFunc {
		return auth.Middleware(jwtSecret, auth.RequireRole([]string{"admin"}, h))
	}
	staffOnly := func(h http.HandlerFunc) http.HandlerFunc {
		return auth.Middleware(jwtSecret, auth.RequireRole([]string{"admin", "volunteer"}, h))
	}

	mux.HandleFunc("POST /api/auth/register", app.Register)
	mux.HandleFunc("POST /api/auth/login", app.Login)
	mux.HandleFunc("GET /api/siret/verify", app.VerifySiret)
	mux.HandleFunc("GET /api/auth/me", protected(app.Me))
	mux.HandleFunc("GET /api/users", adminOnly(app.ListUsers))
	mux.HandleFunc("PUT /api/users/{id}", adminOnly(app.UpdateUser))
	mux.HandleFunc("PATCH /api/users/{id}/status", adminOnly(app.SetUserStatus))
	mux.HandleFunc("DELETE /api/users/{id}", adminOnly(app.DeleteUser))

	mux.HandleFunc("GET /api/dashboard", adminOnly(app.Dashboard))

	mux.HandleFunc("GET /api/profile", protected(app.GetProfile))
	mux.HandleFunc("PUT /api/profile", protected(app.UpdateProfile))
	mux.HandleFunc("PUT /api/profile/password", protected(app.ChangePassword))
	mux.HandleFunc("GET /api/dues", protected(app.DuesInfo))
	mux.HandleFunc("POST /api/profile/dues/checkout", protected(app.CreateDuesCheckout))
	mux.HandleFunc("POST /api/profile/dues/confirm", protected(app.ConfirmDuesPayment))
	mux.HandleFunc("POST /api/stripe/webhook", app.StripeWebhook)

	mux.HandleFunc("GET /api/cities", app.ListCities)

	mux.HandleFunc("GET /api/containers", staffOnly(app.ListContainers))
	mux.HandleFunc("GET /api/containers/{id}", staffOnly(app.GetContainer))
	mux.HandleFunc("GET /api/containers/{id}/products", staffOnly(app.ContainerProducts))
	mux.HandleFunc("POST /api/containers", adminOnly(app.CreateContainer))
	mux.HandleFunc("PUT /api/containers/{id}", adminOnly(app.UpdateContainer))
	mux.HandleFunc("DELETE /api/containers/{id}", adminOnly(app.DeleteContainer))

	mux.HandleFunc("GET /api/merchants", adminOnly(app.ListMerchants))
	mux.HandleFunc("GET /api/merchants/reminders", adminOnly(app.MembershipReminders))
	mux.HandleFunc("GET /api/merchants/{id}", adminOnly(app.GetMerchant))
	mux.HandleFunc("POST /api/merchants", adminOnly(app.CreateMerchant))
	mux.HandleFunc("PUT /api/merchants/{id}", adminOnly(app.UpdateMerchant))
	mux.HandleFunc("DELETE /api/merchants/{id}", adminOnly(app.DeleteMerchant))

	mux.HandleFunc("GET /api/products", staffOnly(app.ListProducts))
	mux.HandleFunc("GET /api/barcodes/{barcode}", staffOnly(app.GetProductByBarcode))
	mux.HandleFunc("GET /api/products/{id}", staffOnly(app.GetProduct))
	mux.HandleFunc("GET /api/products/{id}/barcode-image", staffOnly(app.ProductBarcodeImage))
	mux.HandleFunc("GET /api/products/{id}/movements", staffOnly(app.ListStockMovements))
	mux.HandleFunc("POST /api/products", staffOnly(app.CreateProduct))
	mux.HandleFunc("PUT /api/products/{id}", staffOnly(app.UpdateProduct))
	mux.HandleFunc("DELETE /api/products/{id}", adminOnly(app.DeleteProduct))
	mux.HandleFunc("DELETE /api/products/{id}/images/{imageId}", staffOnly(app.DeleteProductImage))
	mux.HandleFunc("POST /api/products/{id}/stock", staffOnly(app.MoveStock))

	mux.HandleFunc("GET /api/tours", staffOnly(app.ListTours))
	mux.HandleFunc("GET /api/tours/{id}", staffOnly(app.GetTour))
	mux.HandleFunc("POST /api/tours", staffOnly(app.CreateTour))
	mux.HandleFunc("PATCH /api/tours/{id}/status", staffOnly(app.UpdateTourStatus))
	mux.HandleFunc("DELETE /api/tours/{id}", adminOnly(app.DeleteTour))
	mux.HandleFunc("GET /api/tours/{id}/pdf", staffOnly(app.TourDeliveryPDF))

	mux.HandleFunc("GET /api/skills", app.ListSkills)
	mux.HandleFunc("POST /api/volunteers", app.CreateVolunteer)
	mux.HandleFunc("GET /api/volunteers", adminOnly(app.ListVolunteers))
	mux.HandleFunc("PATCH /api/volunteers/{id}/status", adminOnly(app.SetVolunteerStatus))
	mux.HandleFunc("DELETE /api/volunteers/{id}", adminOnly(app.DeleteVolunteer))

	mux.HandleFunc("GET /api/plannings", protected(app.ListPlannings))
	mux.HandleFunc("GET /api/plannings/mine", protected(app.MyPlannings))
	mux.HandleFunc("GET /api/plannings/created", protected(app.MyCreatedEvents))
	mux.HandleFunc("POST /api/events", protected(auth.RequireRole([]string{"admin", "volunteer"}, app.CreatePlanning)))
	mux.HandleFunc("PATCH /api/plannings/{id}/review", adminOnly(app.ReviewPlanning))
	mux.HandleFunc("GET /api/plannings/{id}", protected(app.GetPlanning))
	mux.HandleFunc("POST /api/plannings/{id}/join", protected(app.JoinPlanning))
	mux.HandleFunc("DELETE /api/plannings/{id}/join", protected(app.LeavePlanning))
	mux.HandleFunc("POST /api/plannings", adminOnly(app.CreatePlanning))
	mux.HandleFunc("PUT /api/plannings/{id}", adminOnly(app.UpdatePlanning))
	mux.HandleFunc("DELETE /api/plannings/{id}", adminOnly(app.DeletePlanning))
	mux.HandleFunc("GET /api/plannings/{id}/excel", adminOnly(app.PlanningExcel))

	mux.HandleFunc("GET /api/services", protected(app.ListServices))
	mux.HandleFunc("GET /api/services/mine", protected(app.MyServices))
	mux.HandleFunc("GET /api/services/{id}", protected(app.GetService))
	mux.HandleFunc("POST /api/services/{id}/subscribe", protected(app.SubscribeService))
	mux.HandleFunc("DELETE /api/services/{id}/subscribe", protected(app.UnsubscribeService))
	mux.HandleFunc("POST /api/services", adminOnly(app.CreateService))
	mux.HandleFunc("PUT /api/services/{id}", adminOnly(app.UpdateService))
	mux.HandleFunc("DELETE /api/services/{id}", adminOnly(app.DeleteService))

	mux.HandleFunc("GET /api/collections", staffOnly(app.ListCollections))
	mux.HandleFunc("GET /api/collections/{id}", staffOnly(app.GetCollection))
	mux.HandleFunc("POST /api/collections", staffOnly(app.CreateCollection))
	mux.HandleFunc("PUT /api/collections/{id}", staffOnly(app.UpdateCollection))
	mux.HandleFunc("PATCH /api/collections/{id}/status", staffOnly(app.SetCollectionStatus))
	mux.HandleFunc("PATCH /api/collections/{id}/complete", staffOnly(app.CompleteCollection))
	mux.HandleFunc("DELETE /api/collections/{id}", adminOnly(app.DeleteCollection))

	mux.HandleFunc("GET /api/products/expiring", staffOnly(app.ExpiringProducts))

	mux.HandleFunc("GET /api/donations", adminOnly(app.ListDonations))
	mux.HandleFunc("GET /api/donations/mine", protected(app.MyDonations))
	mux.HandleFunc("POST /api/donations", protected(auth.RequireRole([]string{"merchant", "admin"}, app.CreateDonation)))
	mux.HandleFunc("PATCH /api/donations/{id}/review", adminOnly(app.ReviewDonation))
	mux.HandleFunc("DELETE /api/donations/{id}", protected(app.DeleteDonation))

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	handler := corsMiddleware(mux)
	log.Printf("NO MORE WASTE API listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

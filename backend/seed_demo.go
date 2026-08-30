package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"nomorewaste/internal/auth"
)

func demoImageDataURI(name, color string) string {
	initial := "?"
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		initial = strings.ToUpper(string([]rune(trimmed)[0:1]))
	}
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="480" height="360">`+
		`<defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1">`+
		`<stop offset="0" stop-color="%s"/><stop offset="1" stop-color="#065f46"/></linearGradient></defs>`+
		`<rect width="100%%" height="100%%" fill="url(#g)"/>`+
		`<text x="50%%" y="54%%" font-family="Arial, sans-serif" font-size="180" font-weight="bold" `+
		`fill="rgba(255,255,255,0.92)" text-anchor="middle" dominant-baseline="middle">%s</text></svg>`,
		color, initial)
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg))
}

type demoUser struct {
	email    string
	password string
	name     string
	role     string
}

type demoMerchant struct {
	company   string
	contact   string
	email     string
	phone     string
	address   string
	endOffset int
	status    string
}

type demoProduct struct {
	name          string
	category      string
	barcode       string
	description   string
	quantity      int
	merchantIndex int
	container     string
	shelf         string
	expiryOffset  int
}

type demoVolunteer struct {
	name   string
	email  string
	phone  string
	status string
	skills []string
}

type demoTourItem struct {
	productIndex int
	quantity     int
}

type demoTour struct {
	label       string
	driver      string
	destination string
	dateOffset  int
	status      string
	items       []demoTourItem
}

type demoSlot struct {
	volunteerIndex int
	task           string
	start          string
	end            string
}

type demoPlanning struct {
	dateOffset      int
	title           string
	description     string
	location        string
	start           string
	end             string
	maxParticipants int
	slots           []demoSlot
}

type demoContainer struct {
	city     string
	label    string
	address  string
	capacity int
	status   string
}

func seedDemoData(db *sql.DB) {
	var merchantCount int
	db.QueryRow("SELECT COUNT(*) FROM merchants").Scan(&merchantCount)
	if merchantCount > 0 {
		return
	}
	log.Println("seeding demo data...")

	users := []demoUser{
		{"directeur@nomorewaste.org", "NoMoreWaste2026!", "Camille Directeur", "admin"},
		{"benevole@nomorewaste.org", "benevole123", "Sofiane Bénévole", "volunteer"},
		{"commercant@nomorewaste.org", "commercant123", "Nadia Commerçante", "merchant"},
		{"boulangerie@nomorewaste.org", "boulangerie123", "Marc Boulanger", "merchant"},
		{"adherent@nomorewaste.org", "adherent123", "Lucas Adhérent", "member"},
	}
	today := time.Now()
	cityNames := []string{"Paris", "Marseille", "Nantes", "Limoges"}
	for index, user := range users {
		hash, err := auth.HashPassword(user.password)
		if err != nil {
			continue
		}
		var cityID interface{}
		var resolved int64
		if err := db.QueryRow("SELECT id FROM cities WHERE name = ?", cityNames[index%len(cityNames)]).
			Scan(&resolved); err == nil {
			cityID = resolved
		}
		duesPaid := 1
		membershipEnd := today.AddDate(1, 0, 0).Format("2006-01-02")
		if user.role == "member" {
			duesPaid = 0
			membershipEnd = today.AddDate(0, 0, -20).Format("2006-01-02")
		}
		var companyName interface{}
		var siretValue interface{}
		if user.role == "merchant" {
			if user.email == "commercant@nomorewaste.org" {
				companyName = "MONOPRIX"
				siretValue = "55201802001808"
			} else {
				companyName = "BOULANGERIE DE PARIS"
				siretValue = "35247171800010"
			}
		}
		db.Exec(`INSERT OR IGNORE INTO users (email, password_hash, full_name, role, phone, address, city_id,
			has_paid_dues, membership_end_date, company_name, siret) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			user.email, hash, user.name, user.role, fmt.Sprintf("06%08d", 10203040+index),
			"12 rue de la Solidarité", cityID, duesPaid, membershipEnd, companyName, siretValue)
	}

	dateFrom := func(offset int) string { return today.AddDate(0, 0, offset).Format("2006-01-02") }

	containers := []demoContainer{
		{"Paris", "PAR-A1", "18 rue de la Chapelle, Paris 18e", 400, "active"},
		{"Paris", "PAR-B2", "7 avenue Daumesnil, Paris 12e", 250, "active"},
		{"Paris", "PAR-C3", "45 rue Riquet, Paris 19e", 180, "maintenance"},
		{"Marseille", "MAR-A1", "12 quai du Port, Marseille 2e", 320, "active"},
		{"Marseille", "MAR-B2", "89 boulevard National, Marseille 3e", 200, "active"},
		{"Nantes", "NAN-A1", "5 quai de la Fosse, Nantes", 280, "active"},
		{"Nantes", "NAN-B2", "31 rue de Strasbourg, Nantes", 150, "active"},
		{"Limoges", "LIM-A1", "22 avenue Garibaldi, Limoges", 220, "active"},
		{"Limoges", "LIM-B2", "3 place de la Motte, Limoges", 120, "full"},
	}
	containerIDs := map[string]int64{}
	for _, container := range containers {
		var cityID int64
		if err := db.QueryRow("SELECT id FROM cities WHERE name = ?", container.city).Scan(&cityID); err != nil {
			continue
		}
		result, err := db.Exec(`INSERT INTO containers (city_id, label, address, capacity, status)
			VALUES (?, ?, ?, ?, ?)`, cityID, container.label, container.address, container.capacity, container.status)
		if err != nil {
			continue
		}
		id, _ := result.LastInsertId()
		containerIDs[container.label] = id
	}

	merchants := []demoMerchant{
		{"Boulangerie du Marché", "Jean Petit", "contact@boulangerie-marche.fr", "0145678901", "12 rue des Halles, Paris", 8, "active"},
		{"Primeur Bio Vert", "Sophie Legrand", "sophie@primeur-vert.fr", "0146789012", "5 avenue de la République, Lyon", 20, "active"},
		{"Épicerie Solidaire Nord", "Karim Benali", "karim@epicerie-nord.fr", "0147890123", "34 rue Nationale, Lille", -5, "active"},
		{"Supermarché Le Panier", "Émilie Rousseau", "contact@le-panier.fr", "0148901234", "88 boulevard Voltaire, Marseille", 120, "active"},
		{"Fromagerie des Alpes", "Pierre Blanc", "pierre@fromagerie-alpes.fr", "0149012345", "2 place du Marché, Grenoble", 250, "active"},
		{"Traiteur Bonté", "Aïcha Diallo", "aicha@traiteur-bonte.fr", "0150123456", "17 rue de la Paix, Bordeaux", 45, "inactive"},
	}
	merchantIDs := make([]int64, len(merchants))
	for index, merchant := range merchants {
		end := dateFrom(merchant.endOffset)
		start := today.AddDate(-1, 0, merchant.endOffset).Format("2006-01-02")
		result, err := db.Exec(`INSERT INTO merchants (company_name, contact_name, email, phone, address,
			membership_start, membership_end, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			merchant.company, merchant.contact, merchant.email, merchant.phone, merchant.address,
			start, end, merchant.status)
		if err != nil {
			continue
		}
		merchantIDs[index], _ = result.LastInsertId()
	}
	db.Exec(`UPDATE merchants SET user_id = (SELECT id FROM users WHERE email = 'commercant@nomorewaste.org')
		WHERE company_name = 'Boulangerie du Marché'`)
	db.Exec(`UPDATE merchants SET user_id = (SELECT id FROM users WHERE email = 'boulangerie@nomorewaste.org')
		WHERE company_name = 'Primeur Bio Vert'`)

	products := []demoProduct{
		{"Baguettes tradition", "Boulangerie", "NMW1000000001", "Baguettes de pain fraîches issues des invendus du jour.", 120, 0, "PAR-A1", "A-01", 1},
		{"Pains de campagne", "Boulangerie", "NMW1000000002", "Pains de campagne au levain, croûte épaisse.", 40, 0, "PAR-A1", "A-02", 2},
		{"Pommes Gala", "Fruits & Légumes", "NMW1000000003", "Pommes Gala croquantes, calibre moyen.", 85, 1, "PAR-A1", "B-04", 3},
		{"Carottes bio", "Fruits & Légumes", "NMW1000000004", "Carottes issues de l'agriculture biologique.", 60, 1, "PAR-B2", "B-05", 5},
		{"Salades vertes", "Fruits & Légumes", "NMW1000000005", "Salades fraîches à distribuer rapidement.", 30, 1, "PAR-B2", "B-06", 0},
		{"Conserves de tomates", "Épicerie", "NMW1000000006", "Conserves de tomates pelées, longue conservation.", 200, 2, "MAR-A1", "C-01", 240},
		{"Pâtes complètes", "Épicerie", "NMW1000000007", "Pâtes complètes riches en fibres.", 150, 2, "MAR-A1", "C-02", 300},
		{"Yaourts nature", "Produits laitiers", "NMW1000000008", "Yaourts nature à consommer sous 5 jours.", 96, 3, "MAR-B2", "D-01", 4},
		{"Fromage comté", "Produits laitiers", "NMW1000000009", "Comté affiné 12 mois, à la coupe.", 18, 4, "NAN-A1", "D-02", 45},
		{"Plats cuisinés végétariens", "Traiteur", "NMW1000000010", "Plats préparés végétariens prêts à réchauffer.", 25, 5, "NAN-A1", "E-01", 2},
		{"Riz basmati", "Épicerie", "NMW1000000011", "Riz basmati parfumé, sac de 1 kg.", 110, 3, "NAN-B2", "C-03", 400},
		{"Bananes équitables", "Fruits & Légumes", "NMW1000000012", "Bananes issues du commerce équitable.", 45, 1, "LIM-A1", "B-07", 6},
	}
	categoryColors := map[string]string{
		"Boulangerie":       "#d97706",
		"Fruits & Légumes":  "#16a34a",
		"Épicerie":          "#0d9488",
		"Produits laitiers": "#0ea5e9",
		"Traiteur":          "#8b5cf6",
	}
	productIDs := make([]int64, len(products))
	for index, product := range products {
		var merchantID interface{}
		if product.merchantIndex >= 0 && merchantIDs[product.merchantIndex] != 0 {
			merchantID = merchantIDs[product.merchantIndex]
		}
		var containerID interface{}
		if id, ok := containerIDs[product.container]; ok {
			containerID = id
		}
		result, err := db.Exec(`INSERT INTO products (name, category, barcode, description, quantity, merchant_id,
			container_id, shelf_code, expiration_date) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, product.name,
			product.category, product.barcode, product.description, product.quantity, merchantID, containerID,
			product.shelf, dateFrom(product.expiryOffset))
		if err != nil {
			continue
		}
		productIDs[index], _ = result.LastInsertId()
		color := categoryColors[product.category]
		if color == "" {
			color = "#10b981"
		}
		db.Exec("INSERT INTO product_images (product_id, image) VALUES (?, ?)",
			productIDs[index], demoImageDataURI(product.name, color))
	}

	movements := []struct {
		productIndex int
		kind         string
		quantity     int
		reason       string
	}{
		{0, "in", 150, "Collecte matinale"},
		{0, "out", 30, "Distribution centre-ville"},
		{2, "in", 100, "Don primeur"},
		{2, "out", 15, "Tournée quartier sud"},
		{5, "in", 200, "Réassort épicerie"},
		{7, "out", 24, "Distribution familles"},
	}
	for _, movement := range movements {
		if productIDs[movement.productIndex] == 0 {
			continue
		}
		db.Exec(`INSERT INTO stock_movements (product_id, movement_type, quantity, reason)
			VALUES (?, ?, ?, ?)`, productIDs[movement.productIndex], movement.kind, movement.quantity, movement.reason)
	}

	volunteers := []demoVolunteer{
		{"Marie Lefèvre", "marie.lefevre@mail.fr", "0601020304", "approved", []string{"chauffeur", "manutentionnaire"}},
		{"Thomas Moreau", "thomas.moreau@mail.fr", "0602030405", "approved", []string{"cuisinier", "accueil"}},
		{"Fatima Zahra", "fatima.zahra@mail.fr", "0603040506", "approved", []string{"accueil", "informatique"}},
		{"David Nguyen", "david.nguyen@mail.fr", "0604050607", "approved", []string{"chauffeur"}},
		{"Julie Bernard", "julie.bernard@mail.fr", "0605060708", "pending", []string{"cuisinier"}},
		{"Antoine Dubois", "antoine.dubois@mail.fr", "0606070809", "pending", []string{"plombier", "manutentionnaire"}},
		{"Léa Girard", "lea.girard@mail.fr", "0607080910", "pending", []string{"accueil"}},
		{"Mehdi Haddad", "mehdi.haddad@mail.fr", "0608091011", "rejected", []string{"informatique"}},
	}
	volunteerIDs := make([]int64, len(volunteers))
	for index, volunteer := range volunteers {
		result, err := db.Exec(`INSERT INTO volunteers (full_name, email, phone, status) VALUES (?, ?, ?, ?)`,
			volunteer.name, volunteer.email, volunteer.phone, volunteer.status)
		if err != nil {
			continue
		}
		volunteerIDs[index], _ = result.LastInsertId()
		for _, skillName := range volunteer.skills {
			var skillID int64
			if err := db.QueryRow("SELECT id FROM skills WHERE name = ?", skillName).Scan(&skillID); err != nil {
				continue
			}
			db.Exec("INSERT OR IGNORE INTO volunteer_skills (volunteer_id, skill_id) VALUES (?, ?)",
				volunteerIDs[index], skillID)
		}
	}

	tours := []demoTour{
		{"Tournée Centre-Ville", "Marie Lefèvre", "Centre d'accueil Saint-Vincent", -2, "delivered",
			[]demoTourItem{{0, 40}, {2, 20}, {7, 24}}},
		{"Tournée Quartier Sud", "David Nguyen", "Foyer Les Tilleuls", 1, "planned",
			[]demoTourItem{{5, 50}, {6, 30}, {10, 25}}},
		{"Tournée Écoles", "Marie Lefèvre", "Cantine solidaire municipale", 3, "planned",
			[]demoTourItem{{2, 30}, {4, 15}, {11, 20}}},
		{"Tournée Nord", "David Nguyen", "Épicerie Solidaire Nord", 5, "planned",
			[]demoTourItem{{1, 20}, {8, 10}, {9, 12}}},
	}
	for _, tour := range tours {
		result, err := db.Exec(`INSERT INTO tours (label, driver_name, destination, scheduled_date, status)
			VALUES (?, ?, ?, ?, ?)`, tour.label, tour.driver, tour.destination, dateFrom(tour.dateOffset), tour.status)
		if err != nil {
			continue
		}
		tourID, _ := result.LastInsertId()
		for _, item := range tour.items {
			if productIDs[item.productIndex] == 0 {
				continue
			}
			db.Exec("INSERT INTO tour_items (tour_id, product_id, quantity) VALUES (?, ?, ?)",
				tourID, productIDs[item.productIndex], item.quantity)
		}
	}

	plannings := []demoPlanning{
		{0, "Distribution du jour", "Distribution de colis alimentaires aux familles bénéficiaires.",
			"Centre d'accueil Saint-Vincent, Paris", "08:00", "16:00", 8, []demoSlot{
				{0, "Conduite tournée centre-ville", "08:00", "12:00"},
				{1, "Préparation des colis", "09:00", "13:00"},
				{2, "Accueil des bénéficiaires", "10:00", "16:00"},
			}},
		{2, "Collecte & tri des invendus", "Collecte chez les commerçants partenaires puis tri au conteneur.",
			"Conteneur PAR-A1, Paris 18e", "07:30", "18:00", 6, []demoSlot{
				{3, "Collecte chez les commerçants", "07:30", "11:30"},
				{0, "Tri et rangement du stock", "13:00", "17:00"},
			}},
		{5, "Atelier cuisine solidaire", "Préparation de repas chauds à partir des invendus collectés.",
			"Cuisine associative, Marseille", "10:00", "15:00", 12, []demoSlot{
				{1, "Animation atelier cuisine", "10:00", "15:00"},
			}},
		{9, "Maraude et distribution", "Distribution mobile auprès des personnes en situation de rue.",
			"Départ conteneur NAN-A1, Nantes", "17:00", "21:00", 10, []demoSlot{
				{2, "Préparation de la maraude", "17:00", "18:30"},
			}},
		{14, "Grande collecte trimestrielle", "Collecte exceptionnelle dans les supermarchés partenaires.",
			"Limoges centre", "09:00", "18:00", 20, []demoSlot{}},
	}
	for index, planning := range plannings {
		result, err := db.Exec(`INSERT INTO plannings (planning_date, title, description, location,
			start_time, end_time, max_participants) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			dateFrom(planning.dateOffset), planning.title, planning.description, planning.location,
			planning.start, planning.end, planning.maxParticipants)
		if err != nil {
			continue
		}
		planningID, _ := result.LastInsertId()
		for _, slot := range planning.slots {
			if volunteerIDs[slot.volunteerIndex] == 0 {
				continue
			}
			db.Exec(`INSERT INTO planning_slots (planning_id, volunteer_id, task, start_time, end_time)
				VALUES (?, ?, ?, ?, ?)`, planningID, volunteerIDs[slot.volunteerIndex], slot.task, slot.start, slot.end)
		}
		if index < 3 {
			rows, err := db.Query("SELECT id FROM users WHERE role IN ('volunteer', 'member') LIMIT ?", index+1)
			if err != nil {
				continue
			}
			userIDs := []int64{}
			for rows.Next() {
				var userID int64
				if err := rows.Scan(&userID); err == nil {
					userIDs = append(userIDs, userID)
				}
			}
			rows.Close()
			for _, userID := range userIDs {
				db.Exec("INSERT OR IGNORE INTO planning_participants (planning_id, user_id) VALUES (?, ?)",
					planningID, userID)
			}
		}
	}

	seedServicesAndCollections(db, volunteerIDs, merchantIDs, today)
	seedDonationsAndEvents(db, today)

	log.Println("demo data seeded")
}

type demoService struct {
	title       string
	category    string
	description string
	dayOffset   int
	hour        string
	location    string
	capacity    int
}

type demoCollection struct {
	label         string
	driverIndex   int
	dayOffset     int
	status        string
	merchantOrder []int
}

func seedServicesAndCollections(db *sql.DB, volunteerIDs []int64, merchantIDs []int64, today time.Time) {
	dateFrom := func(offset int) string { return today.AddDate(0, 0, offset).Format("2006-01-02") }

	services := []demoService{
		{"Atelier cuisine anti-gaspi", "cuisine", "Apprenez à cuisiner les invendus et à limiter les pertes.", 3, "14:00", "Cuisine associative, Paris 18e", 12},
		{"Initiation au bricolage", "bricolage", "Petites réparations du quotidien : perceuse, vis et chevilles.", 6, "10:00", "Atelier NMW, Nantes", 8},
		{"Dépannage électrique", "electricite", "Diagnostic et remise aux normes de vos petites installations.", 8, "09:30", "Local technique, Marseille", 6},
		{"Permanence plomberie", "plomberie", "Réparation de fuites et remplacement de joints.", 10, "15:00", "Local technique, Limoges", 5},
		{"Réparation petit électroménager", "reparation", "Donnez une seconde vie à vos appareils.", 12, "13:30", "Repair café, Paris 12e", 10},
		{"Entretien véhicule", "vehicule", "Vérification et petit entretien de votre voiture.", 15, "09:00", "Garage solidaire, Nantes", 6},
		{"Gardiennage solidaire", "gardiennage", "Service de garde ponctuel entre adhérents.", 18, "08:00", "Siège NMW, Paris", 4},
	}
	for _, service := range services {
		db.Exec(`INSERT INTO services (title, category, description, date_time, location, max_capacity, status)
			VALUES (?, ?, ?, ?, ?, ?, 'open')`, service.title, service.category, service.description,
			dateFrom(service.dayOffset)+" "+service.hour, service.location, service.capacity)
	}

	collections := []demoCollection{
		{"Ramassage Paris Nord", 0, 1, "planned", []int{0, 1}},
		{"Ramassage Marseille Centre", 3, 2, "planned", []int{3, 5}},
		{"Ramassage Nantes", 0, 4, "in_progress", []int{2, 4}},
	}
	for _, collection := range collections {
		var driverID interface{}
		if collection.driverIndex < len(volunteerIDs) && volunteerIDs[collection.driverIndex] != 0 {
			driverID = volunteerIDs[collection.driverIndex]
		}
		result, err := db.Exec(`INSERT INTO collections (driver_id, label, scheduled_date, status)
			VALUES (?, ?, ?, ?)`, driverID, collection.label, dateFrom(collection.dayOffset), collection.status)
		if err != nil {
			continue
		}
		collectionID, _ := result.LastInsertId()
		for index, merchantIndex := range collection.merchantOrder {
			if merchantIndex >= len(merchantIDs) || merchantIDs[merchantIndex] == 0 {
				continue
			}
			db.Exec(`INSERT INTO collection_stops (collection_id, merchant_id, order_index)
				VALUES (?, ?, ?)`, collectionID, merchantIDs[merchantIndex], index+1)
		}
	}
}

func seedDonationsAndEvents(db *sql.DB, today time.Time) {
	dateFrom := func(offset int) string { return today.AddDate(0, 0, offset).Format("2006-01-02") }

	donations := []struct {
		email        string
		title        string
		kind         string
		category     string
		description  string
		quantity     int
		expiryOffset int
		fromOffset   int
		status       string
	}{
		{"commercant@nomorewaste.org", "Invendus fruits et légumes", "food", "Fruits & Légumes",
			"Cagettes de fruits et légumes du jour, encore parfaitement consommables.", 40, 3, 1, "pending"},
		{"commercant@nomorewaste.org", "Produits secs longue conservation", "food", "Épicerie",
			"Pâtes, riz et conserves proches de la date de vente optimale.", 120, 200, 2, "pending"},
		{"boulangerie@nomorewaste.org", "Pains et viennoiseries du soir", "food", "Boulangerie",
			"Production non vendue en fin de journée, à récupérer avant 20h.", 60, 1, 1, "pending"},
		{"boulangerie@nomorewaste.org", "Cartons et cagettes réutilisables", "object", "Matériel",
			"Cagettes en bois et cartons propres pour le transport des denrées.", 25, 0, 3, "pending"},
	}
	for _, donation := range donations {
		var userID int64
		if err := db.QueryRow("SELECT id FROM users WHERE email = ?", donation.email).Scan(&userID); err != nil {
			continue
		}
		var merchantID interface{}
		var resolved int64
		if err := db.QueryRow("SELECT id FROM merchants WHERE user_id = ?", userID).Scan(&resolved); err == nil {
			merchantID = resolved
		}
		var expiration interface{}
		if donation.kind == "food" {
			expiration = dateFrom(donation.expiryOffset)
		}
		db.Exec(`INSERT INTO donation_offers (user_id, merchant_id, title, donation_type, category,
			description, quantity, expiration_date, pickup_address, available_from, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, merchantID, donation.title, donation.kind, donation.category, donation.description,
			donation.quantity, expiration, "12 rue de la Solidarité", dateFrom(donation.fromOffset),
			donation.status)
	}

	var volunteerUserID int64
	if err := db.QueryRow("SELECT id FROM users WHERE email = 'benevole@nomorewaste.org'").
		Scan(&volunteerUserID); err != nil {
		return
	}
	events := []struct {
		title       string
		description string
		location    string
		dayOffset   int
		start       string
		end         string
		capacity    int
		eventType   string
		status      string
	}{
		{"Brocante solidaire de printemps", "Vide-grenier au profit de l association, stands tenus par les benevoles.",
			"Place de la Republique, Paris 11e", 21, "09:00", "18:00", 40, "brocante", "pending"},
		{"Collecte devant le supermarche", "Operation de collecte alimentaire a l entree du magasin partenaire.",
			"Monoprix Clichy", 12, "10:00", "19:00", 15, "collecte", "pending"},
		{"Atelier reparation velos", "Remise en etat de velos donnes puis redistribution aux beneficiaires.",
			"Atelier NMW, Nantes", 28, "14:00", "18:00", 12, "atelier", "approved"},
	}
	for _, event := range events {
		db.Exec(`INSERT INTO plannings (planning_date, title, description, location, start_time, end_time,
			max_participants, event_type, created_by, approval_status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			dateFrom(event.dayOffset), event.title, event.description, event.location, event.start,
			event.end, event.capacity, event.eventType, volunteerUserID, event.status)
	}
}

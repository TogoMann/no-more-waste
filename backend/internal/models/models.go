package models

type User struct {
	ID                int64  `json:"id"`
	Email             string `json:"email"`
	PasswordHash      string `json:"-"`
	FullName          string `json:"full_name"`
	Role              string `json:"role"`
	Status            string `json:"status"`
	Phone             string `json:"phone"`
	Address           string `json:"address"`
	CityID            *int64 `json:"city_id"`
	CityName          string `json:"city_name,omitempty"`
	MembershipEndDate string `json:"membership_end_date"`
	HasPaidDues       bool   `json:"has_paid_dues"`
	MembershipValid   bool   `json:"membership_valid"`
	CompanyName       string `json:"company_name"`
	Siret             string `json:"siret"`
	CreatedAt         string `json:"created_at"`
}

type City struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Container struct {
	ID        int64  `json:"id"`
	CityID    int64  `json:"city_id"`
	CityName  string `json:"city_name,omitempty"`
	Label     string `json:"label"`
	Address   string `json:"address"`
	Capacity  int    `json:"capacity"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	Stored    int    `json:"stored"`
	Products  int    `json:"products"`
	Occupancy int    `json:"occupancy"`
}

type Merchant struct {
	ID              int64  `json:"id"`
	UserID          *int64 `json:"user_id"`
	CompanyName     string `json:"company_name"`
	ContactName     string `json:"contact_name"`
	Email           string `json:"email"`
	Phone           string `json:"phone"`
	Address         string `json:"address"`
	MembershipStart string `json:"membership_start"`
	MembershipEnd   string `json:"membership_end"`
	Status          string `json:"status"`
	CreatedAt       string `json:"created_at"`
}

type Product struct {
	ID             int64          `json:"id"`
	Name           string         `json:"name"`
	Category       string         `json:"category"`
	Barcode        string         `json:"barcode"`
	Description    string         `json:"description"`
	Quantity       int            `json:"quantity"`
	MerchantID     *int64         `json:"merchant_id"`
	ContainerID    *int64         `json:"container_id"`
	ContainerName  string         `json:"container_name,omitempty"`
	CityName       string         `json:"city_name,omitempty"`
	ShelfCode      string         `json:"shelf_code"`
	ExpirationDate string         `json:"expiration_date"`
	DaysToExpiry   *int           `json:"days_to_expiry,omitempty"`
	CreatedAt      string         `json:"created_at"`
	Thumbnail      string         `json:"thumbnail,omitempty"`
	ImageCount     int            `json:"image_count"`
	Images         []ProductImage `json:"images,omitempty"`
}

type ProductImage struct {
	ID    int64  `json:"id"`
	Image string `json:"image"`
}

type StockMovement struct {
	ID           int64  `json:"id"`
	ProductID    int64  `json:"product_id"`
	MovementType string `json:"movement_type"`
	Quantity     int    `json:"quantity"`
	Reason       string `json:"reason"`
	CreatedBy    *int64 `json:"created_by"`
	CreatedAt    string `json:"created_at"`
}

type Tour struct {
	ID            int64      `json:"id"`
	Label         string     `json:"label"`
	DriverName    string     `json:"driver_name"`
	Destination   string     `json:"destination"`
	ScheduledDate string     `json:"scheduled_date"`
	Status        string     `json:"status"`
	CreatedAt     string     `json:"created_at"`
	Items         []TourItem `json:"items,omitempty"`
}

type TourItem struct {
	ID          int64  `json:"id"`
	TourID      int64  `json:"tour_id"`
	ProductID   int64  `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int    `json:"quantity"`
}

type Skill struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Volunteer struct {
	ID        int64   `json:"id"`
	UserID    *int64  `json:"user_id"`
	FullName  string  `json:"full_name"`
	Email     string  `json:"email"`
	Phone     string  `json:"phone"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
	Skills    []Skill `json:"skills,omitempty"`
}

type Planning struct {
	ID               int64          `json:"id"`
	PlanningDate     string         `json:"planning_date"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	Location         string         `json:"location"`
	StartTime        string         `json:"start_time"`
	EndTime          string         `json:"end_time"`
	MaxParticipants  int            `json:"max_participants"`
	EventType        string         `json:"event_type"`
	CreatedBy        *int64         `json:"created_by"`
	CreatorName      string         `json:"creator_name,omitempty"`
	ApprovalStatus   string         `json:"approval_status"`
	ReviewNote       string         `json:"review_note"`
	ParticipantCount int            `json:"participant_count"`
	Joined           bool           `json:"joined"`
	CreatedAt        string         `json:"created_at"`
	Slots            []PlanningSlot `json:"slots,omitempty"`
	Participants     []Participant  `json:"participants,omitempty"`
}

type Participant struct {
	UserID   int64  `json:"user_id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	JoinedAt string `json:"joined_at"`
}

type PlanningSlot struct {
	ID            int64  `json:"id"`
	PlanningID    int64  `json:"planning_id"`
	VolunteerID   int64  `json:"volunteer_id"`
	VolunteerName string `json:"volunteer_name"`
	Task          string `json:"task"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
}

type Service struct {
	ID              int64         `json:"id"`
	Title           string        `json:"title"`
	Category        string        `json:"category"`
	Description     string        `json:"description"`
	DateTime        string        `json:"date_time"`
	Location        string        `json:"location"`
	MaxCapacity     int           `json:"max_capacity"`
	Status          string        `json:"status"`
	CreatedAt       string        `json:"created_at"`
	SubscriberCount int           `json:"subscriber_count"`
	Subscribed      bool          `json:"subscribed"`
	Subscribers     []Participant `json:"subscribers,omitempty"`
}

type Collection struct {
	ID            int64            `json:"id"`
	DriverID      *int64           `json:"driver_id"`
	DriverName    string           `json:"driver_name"`
	Label         string           `json:"label"`
	ScheduledDate string           `json:"scheduled_date"`
	Status        string           `json:"status"`
	CompletedAt   string           `json:"completed_at"`
	CreatedAt     string           `json:"created_at"`
	Stops         []CollectionStop `json:"stops,omitempty"`
	StopCount     int              `json:"stop_count"`
}

type CollectionStop struct {
	ID           int64  `json:"id"`
	CollectionID int64  `json:"collection_id"`
	MerchantID   int64  `json:"merchant_id"`
	MerchantName string `json:"merchant_name"`
	Address      string `json:"address"`
	OrderIndex   int    `json:"order_index"`
	Collected    bool   `json:"collected"`
}

type DonationOffer struct {
	ID             int64  `json:"id"`
	UserID         int64  `json:"user_id"`
	MerchantID     *int64 `json:"merchant_id"`
	DonorName      string `json:"donor_name,omitempty"`
	CompanyName    string `json:"company_name,omitempty"`
	Title          string `json:"title"`
	DonationType   string `json:"donation_type"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	ExpirationDate string `json:"expiration_date"`
	PickupAddress  string `json:"pickup_address"`
	AvailableFrom  string `json:"available_from"`
	Status         string `json:"status"`
	ReviewNote     string `json:"review_note"`
	CollectionID   *int64 `json:"collection_id"`
	CollectionDate string `json:"collection_date,omitempty"`
	CreatedAt      string `json:"created_at"`
}

type Payment struct {
	ID          int64  `json:"id"`
	UserID      int64  `json:"user_id"`
	SessionID   string `json:"session_id"`
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	PaidAt      string `json:"paid_at"`
}

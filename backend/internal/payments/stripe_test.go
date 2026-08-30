package payments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"
)

func testClient(t *testing.T) *Client {
	key := os.Getenv("STRIPE_SECRET_KEY")
	if key == "" {
		t.Skip("STRIPE_SECRET_KEY absent")
	}
	return NewClient(key, os.Getenv("STRIPE_PUBLIC_KEY"), "whsec_test")
}

func TestEnabled(t *testing.T) {
	if NewClient("", "", "").Enabled() {
		t.Fatal("un client sans cle ne doit pas etre actif")
	}
	if !NewClient("sk_test_x", "", "").Enabled() {
		t.Fatal("un client avec cle sk_ doit etre actif")
	}
}

func TestCheckoutRoundTrip(t *testing.T) {
	client := testClient(t)
	session, err := client.CreateCheckoutSession(CheckoutRequest{
		AmountCents: 2000,
		ProductName: "Cotisation annuelle",
		ClientRef:   "42",
		SuccessURL:  "http://localhost:8081/espace/profil?session_id={CHECKOUT_SESSION_ID}",
		CancelURL:   "http://localhost:8081/espace/profil",
	})
	if err != nil {
		t.Fatalf("creation session: %v", err)
	}
	if session.URL == "" || session.ID == "" {
		t.Fatal("session incomplete")
	}
	fetched, err := client.RetrieveCheckoutSession(session.ID)
	if err != nil {
		t.Fatalf("relecture session: %v", err)
	}
	if fetched.PaymentStatus != "unpaid" {
		t.Fatalf("statut attendu unpaid, obtenu %s", fetched.PaymentStatus)
	}
	if fetched.ClientReference != "42" {
		t.Fatalf("client_reference_id perdu: %q", fetched.ClientReference)
	}
	t.Logf("session=%s status=%s montant=%d", fetched.ID, fetched.PaymentStatus, fetched.AmountTotal)
}

func TestVerifyWebhook(t *testing.T) {
	client := NewClient("sk_test_x", "", "whsec_secret")
	payload := []byte(`{"id":"evt_1","type":"checkout.session.completed","data":{"object":{"id":"cs_1","payment_status":"paid","client_reference_id":"7"}}}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	mac := hmac.New(sha256.New, []byte("whsec_secret"))
	mac.Write([]byte(timestamp + "." + string(payload)))
	header := "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))

	event, err := client.VerifyWebhook(payload, header)
	if err != nil {
		t.Fatalf("signature valide rejetee: %v", err)
	}
	if event.Data.Object.ClientReference != "7" {
		t.Fatalf("reference perdue: %q", event.Data.Object.ClientReference)
	}
	if _, err := client.VerifyWebhook(payload, "t="+timestamp+",v1=deadbeef"); err == nil {
		t.Fatal("une signature invalide doit etre rejetee")
	}
}

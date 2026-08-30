package payments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("stripe n est pas configure")
var ErrInvalidSignature = errors.New("signature de webhook invalide")

type Client struct {
	SecretKey     string
	PublicKey     string
	WebhookSecret string
	HTTPClient    *http.Client
}

func NewClient(secretKey, publicKey, webhookSecret string) *Client {
	return &Client{
		SecretKey:     secretKey,
		PublicKey:     publicKey,
		WebhookSecret: webhookSecret,
		HTTPClient:    &http.Client{Timeout: 12 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c != nil && strings.HasPrefix(c.SecretKey, "sk_")
}

type CheckoutSession struct {
	ID              string `json:"id"`
	URL             string `json:"url"`
	Status          string `json:"status"`
	PaymentStatus   string `json:"payment_status"`
	AmountTotal     int64  `json:"amount_total"`
	Currency        string `json:"currency"`
	ClientReference string `json:"client_reference_id"`
	PaymentIntent   string `json:"payment_intent"`
	CustomerEmail   string `json:"customer_email"`
}

type CheckoutRequest struct {
	AmountCents   int64
	Currency      string
	ProductName   string
	Description   string
	CustomerEmail string
	ClientRef     string
	SuccessURL    string
	CancelURL     string
}

func (c *Client) post(path string, form url.Values, target interface{}) error {
	request, err := http.NewRequest(http.MethodPost, "https://api.stripe.com/v1/"+path,
		strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	request.SetBasicAuth(c.SecretKey, "")
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		var failure struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		json.NewDecoder(response.Body).Decode(&failure)
		if failure.Error.Message != "" {
			return errors.New(failure.Error.Message)
		}
		return fmt.Errorf("stripe a repondu %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func (c *Client) get(path string, target interface{}) error {
	request, err := http.NewRequest(http.MethodGet, "https://api.stripe.com/v1/"+path, nil)
	if err != nil {
		return err
	}
	request.SetBasicAuth(c.SecretKey, "")
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("stripe a repondu %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func (c *Client) CreateCheckoutSession(request CheckoutRequest) (*CheckoutSession, error) {
	if !c.Enabled() {
		return nil, ErrNotConfigured
	}
	if request.Currency == "" {
		request.Currency = "eur"
	}
	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", request.SuccessURL)
	form.Set("cancel_url", request.CancelURL)
	form.Set("client_reference_id", request.ClientRef)
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", request.Currency)
	form.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(request.AmountCents, 10))
	form.Set("line_items[0][price_data][product_data][name]", request.ProductName)
	if request.Description != "" {
		form.Set("line_items[0][price_data][product_data][description]", request.Description)
	}
	if request.CustomerEmail != "" {
		form.Set("customer_email", request.CustomerEmail)
	}
	session := &CheckoutSession{}
	if err := c.post("checkout/sessions", form, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (c *Client) RetrieveCheckoutSession(sessionID string) (*CheckoutSession, error) {
	if !c.Enabled() {
		return nil, ErrNotConfigured
	}
	session := &CheckoutSession{}
	if err := c.get("checkout/sessions/"+url.PathEscape(sessionID), session); err != nil {
		return nil, err
	}
	return session, nil
}

type WebhookEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object CheckoutSession `json:"object"`
	} `json:"data"`
}

func (c *Client) VerifyWebhook(payload []byte, signatureHeader string) (*WebhookEvent, error) {
	if c.WebhookSecret == "" {
		return nil, ErrNotConfigured
	}
	timestamp := ""
	signatures := []string{}
	for _, part := range strings.Split(signatureHeader, ",") {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) != 2 {
			continue
		}
		if pair[0] == "t" {
			timestamp = pair[1]
		}
		if pair[0] == "v1" {
			signatures = append(signatures, pair[1])
		}
	}
	if timestamp == "" || len(signatures) == 0 {
		return nil, ErrInvalidSignature
	}
	mac := hmac.New(sha256.New, []byte(c.WebhookSecret))
	mac.Write([]byte(timestamp + "." + string(payload)))
	expected := hex.EncodeToString(mac.Sum(nil))
	matched := false
	for _, signature := range signatures {
		if hmac.Equal([]byte(signature), []byte(expected)) {
			matched = true
			break
		}
	}
	if !matched {
		return nil, ErrInvalidSignature
	}
	event := &WebhookEvent{}
	if err := json.Unmarshal(payload, event); err != nil {
		return nil, err
	}
	return event, nil
}

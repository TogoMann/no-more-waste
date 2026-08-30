package exports

import (
	"strings"
	"testing"

	"nomorewaste/internal/models"
)

func TestGenerateBarcodeValue(t *testing.T) {
	value := GenerateBarcodeValue()
	if !strings.HasPrefix(value, "NMW") {
		t.Fatalf("expected NMW prefix, got %s", value)
	}
}

func TestBarcodePNGBase64(t *testing.T) {
	image, err := BarcodePNGBase64("NMW0000000001")
	if err != nil {
		t.Fatalf("barcode error: %v", err)
	}
	if !strings.HasPrefix(image, "data:image/png;base64,") {
		t.Fatal("expected data uri png")
	}
}

func TestTourDeliveryPDF(t *testing.T) {
	tour := models.Tour{
		Label:       "T1",
		Destination: "Paris",
		Items:       []models.TourItem{{ProductID: 1, ProductName: "Pain", Quantity: 5}},
	}
	data, err := TourDeliveryPDF(tour)
	if err != nil {
		t.Fatalf("pdf error: %v", err)
	}
	if len(data) < 4 || string(data[:4]) != "%PDF" {
		t.Fatal("expected pdf magic header")
	}
}

func TestPlanningExcel(t *testing.T) {
	planning := models.Planning{
		Title:        "Jour 1",
		PlanningDate: "2026-08-01",
		Slots:        []models.PlanningSlot{{VolunteerName: "Marie", Task: "Tri", StartTime: "09:00", EndTime: "12:00"}},
	}
	data, err := PlanningExcel(planning)
	if err != nil {
		t.Fatalf("excel error: %v", err)
	}
	if len(data) < 2 || data[0] != 'P' || data[1] != 'K' {
		t.Fatal("expected xlsx zip header")
	}
}

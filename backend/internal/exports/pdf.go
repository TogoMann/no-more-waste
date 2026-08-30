package exports

import (
	"bytes"

	"github.com/phpdave11/gofpdf"
	"nomorewaste/internal/models"
)

func TourDeliveryPDF(tour models.Tour) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	pdf.SetFont("Arial", "B", 18)
	pdf.Cell(0, 12, "NO MORE WASTE")
	pdf.Ln(12)
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(0, 10, "Recapitulatif de livraison")
	pdf.Ln(14)

	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, "Tournee: "+tour.Label)
	pdf.Ln(7)
	pdf.Cell(0, 7, "Destination: "+tour.Destination)
	pdf.Ln(7)
	pdf.Cell(0, 7, "Chauffeur: "+tour.DriverName)
	pdf.Ln(7)
	pdf.Cell(0, 7, "Date prevue: "+tour.ScheduledDate)
	pdf.Ln(7)
	pdf.Cell(0, 7, "Statut: "+tour.Status)
	pdf.Ln(12)

	pdf.SetFont("Arial", "B", 11)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(20, 8, "Ref", "1", 0, "C", true, 0, "")
	pdf.CellFormat(120, 8, "Produit", "1", 0, "L", true, 0, "")
	pdf.CellFormat(40, 8, "Quantite", "1", 1, "C", true, 0, "")

	pdf.SetFont("Arial", "", 11)
	totalQuantity := 0
	for _, item := range tour.Items {
		pdf.CellFormat(20, 8, itoa(item.ProductID), "1", 0, "C", false, 0, "")
		pdf.CellFormat(120, 8, item.ProductName, "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 8, itoa(int64(item.Quantity)), "1", 1, "C", false, 0, "")
		totalQuantity += item.Quantity
	}

	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(140, 8, "Total", "1", 0, "R", false, 0, "")
	pdf.CellFormat(40, 8, itoa(int64(totalQuantity)), "1", 1, "C", false, 0, "")

	pdf.Ln(20)
	pdf.SetFont("Arial", "", 11)
	pdf.Cell(0, 7, "Signature chauffeur: ____________________")
	pdf.Ln(10)
	pdf.Cell(0, 7, "Signature destinataire: ____________________")

	var buffer bytes.Buffer
	if err := pdf.Output(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func itoa(value int64) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := []byte{}
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

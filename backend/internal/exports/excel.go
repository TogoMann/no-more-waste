package exports

import (
	"bytes"

	"github.com/xuri/excelize/v2"
	"nomorewaste/internal/models"
)

func PlanningExcel(planning models.Planning) ([]byte, error) {
	file := excelize.NewFile()
	sheet := "Planning"
	file.SetSheetName(file.GetSheetName(0), sheet)

	file.SetCellValue(sheet, "A1", "NO MORE WASTE - Planning benevoles")
	file.MergeCell(sheet, "A1", "D1")
	file.SetCellValue(sheet, "A2", "Titre")
	file.SetCellValue(sheet, "B2", planning.Title)
	file.SetCellValue(sheet, "A3", "Date")
	file.SetCellValue(sheet, "B3", planning.PlanningDate)

	headerRow := 5
	file.SetCellValue(sheet, "A5", "Benevole")
	file.SetCellValue(sheet, "B5", "Tache")
	file.SetCellValue(sheet, "C5", "Debut")
	file.SetCellValue(sheet, "D5", "Fin")

	headerStyle, _ := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"2E7D32"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	file.SetCellStyle(sheet, "A5", "D5", headerStyle)

	row := headerRow + 1
	for _, slot := range planning.Slots {
		file.SetCellValue(sheet, cell("A", row), slot.VolunteerName)
		file.SetCellValue(sheet, cell("B", row), slot.Task)
		file.SetCellValue(sheet, cell("C", row), slot.StartTime)
		file.SetCellValue(sheet, cell("D", row), slot.EndTime)
		row++
	}

	file.SetColWidth(sheet, "A", "A", 30)
	file.SetColWidth(sheet, "B", "B", 30)
	file.SetColWidth(sheet, "C", "D", 12)

	var buffer bytes.Buffer
	if err := file.Write(&buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func cell(column string, row int) string {
	digits := []byte{}
	value := row
	for value > 0 {
		digits = append([]byte{byte('0' + value%10)}, digits...)
		value /= 10
	}
	return column + string(digits)
}

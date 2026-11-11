package server

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const (
	maxExcelSize = 10 << 20 // 10 MiB
	maxExcelRows = 5000
)

type importResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Total    int      `json:"total"`
	Errors   []string `json:"errors"`
}

type importColumns struct {
	Category int
	Company  int
	Link     int
	Status   int
}

var categoryAliases = map[string]string{
	"spring":        models.CategorySpring,
	"springrecruit": models.CategorySpring,
	"springhiring":  models.CategorySpring,
	"春招":            models.CategorySpring,
	"春季招聘":          models.CategorySpring,
	"春採用":           models.CategorySpring,
	"봄채용":           models.CategorySpring,
	"봄":             models.CategorySpring,
	"autumn":        models.CategoryAutumn,
	"fall":          models.CategoryAutumn,
	"autumnrecruit": models.CategoryAutumn,
	"autumnhiring":  models.CategoryAutumn,
	"秋招":            models.CategoryAutumn,
	"秋季招聘":          models.CategoryAutumn,
	"秋採用":           models.CategoryAutumn,
	"가을채용":          models.CategoryAutumn,
	"가을":            models.CategoryAutumn,
	"internship":    models.CategoryInternship,
	"intern":        models.CategoryInternship,
	"实习":            models.CategoryInternship,
	"實習":            models.CategoryInternship,
	"インターン":         models.CategoryInternship,
	"インターンシップ":      models.CategoryInternship,
	"인턴":            models.CategoryInternship,
	"인턴십":           models.CategoryInternship,
}

var statusAliases = map[string]string{
	"screening":   models.StatusScreening,
	"screen":      models.StatusScreening,
	"初筛":          models.StatusScreening,
	"初篩":          models.StatusScreening,
	"简历筛选":        models.StatusScreening,
	"簡歷篩選":        models.StatusScreening,
	"書類選考":        models.StatusScreening,
	"서류":          models.StatusScreening,
	"서류전형":        models.StatusScreening,
	"written":     models.StatusWritten,
	"writtentest": models.StatusWritten,
	"test":        models.StatusWritten,
	"笔试":          models.StatusWritten,
	"筆試":          models.StatusWritten,
	"筆記試験":        models.StatusWritten,
	"필기":          models.StatusWritten,
	"필기시험":        models.StatusWritten,
	"interview":   models.StatusInterview,
	"面试":          models.StatusInterview,
	"面試":          models.StatusInterview,
	"面接":          models.StatusInterview,
	"면접":          models.StatusInterview,
	"offer":       models.StatusOffer,
	"offered":     models.StatusOffer,
	"录用":          models.StatusOffer,
	"錄用":          models.StatusOffer,
	"内定":          models.StatusOffer,
	"오퍼":          models.StatusOffer,
	"합격":          models.StatusOffer,
	"rejected":    models.StatusRejected,
	"reject":      models.StatusRejected,
	"淘汰":          models.StatusRejected,
	"拒绝":          models.StatusRejected,
	"拒絕":          models.StatusRejected,
	"不合格":         models.StatusRejected,
	"탈락":          models.StatusRejected,
	"불합격":         models.StatusRejected,
}

func (s *Server) downloadTemplate(c *gin.Context) {
	file, err := buildTemplate()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "template_failed", "Failed to build the Excel template.")
		return
	}
	defer func() {
		_ = file.Close()
	}()
	data, err := file.WriteToBuffer()
	if err != nil {
		apiError(c, http.StatusInternalServerError, "template_failed", "Failed to build the Excel template.")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="resume-trail-template.xlsx"`)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data.Bytes())
}

func buildTemplate() (*excelize.File, error) {
	file := excelize.NewFile()

	const dataSheet = "Applications"
	const helpSheet = "Instructions"
	file.SetSheetName("Sheet1", dataSheet)
	_, _ = file.NewSheet(helpSheet)

	headerStyle, err := file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4F46E5"}},
		Alignment: &excelize.Alignment{Vertical: "center", Horizontal: "left"},
	})
	if err != nil {
		return nil, err
	}
	header := []string{
		"Category / 大分类",
		"Company / 企业名称",
		"Link / 投递链接",
		"Status / 状态",
	}
	for index, value := range header {
		cell, _ := excelize.CoordinatesToCellName(index+1, 1)
		_ = file.SetCellValue(dataSheet, cell, value)
		_ = file.SetCellStyle(dataSheet, cell, cell, headerStyle)
	}
	_ = file.SetRowHeight(dataSheet, 1, 26)
	_ = file.SetColWidth(dataSheet, "A", "A", 24)
	_ = file.SetColWidth(dataSheet, "B", "B", 32)
	_ = file.SetColWidth(dataSheet, "C", "C", 46)
	_ = file.SetColWidth(dataSheet, "D", "D", 20)
	_ = file.SetPanes(dataSheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	categoryList := "spring,autumn,internship,春招,秋招,实习"
	statusList := "screening,written,interview,offer,rejected,初筛,笔试,面试,Offer,淘汰"
	_ = file.AddDataValidation(dataSheet, &excelize.DataValidation{
		Sqref:      "A2:A5000",
		Type:       "list",
		Formula1:   `"` + categoryList + `"`,
		AllowBlank: true,
	})
	_ = file.AddDataValidation(dataSheet, &excelize.DataValidation{
		Sqref:      "D2:D5000",
		Type:       "list",
		Formula1:   `"` + statusList + `"`,
		AllowBlank: true,
	})

	helpRows := [][]interface{}{
		{"ResumeTrail Excel Import / Excel 导入说明"},
		{""},
		{"1. Fill in the Applications sheet. One application per row."},
		{"2. Category accepts spring / autumn / internship or 春招 / 秋招 / 实习 (and other supported languages)."},
		{"3. Status accepts screening / written / interview / offer / rejected or 初筛 / 笔试 / 面试 / Offer / 淘汰."},
		{"4. Company name is required. Link is optional and is saved exactly as entered."},
		{"5. Empty category defaults to spring; empty status defaults to screening."},
		{""},
		{"示例：", ""},
		{"Category", "Company", "Link", "Status"},
		{"春招", "示例科技", "https://example.com/careers", "初筛"},
		{"Autumn", "Example Inc.", "https://example.com/jobs", "Interview"},
	}
	for rowIndex, row := range helpRows {
		for colIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
			_ = file.SetCellValue(helpSheet, cell, value)
		}
	}
	_ = file.SetColWidth(helpSheet, "A", "A", 28)
	_ = file.SetColWidth(helpSheet, "B", "B", 28)
	_ = file.SetColWidth(helpSheet, "C", "C", 42)
	_ = file.SetColWidth(helpSheet, "D", "D", 18)
	titleStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16, Color: "1E1B4B"},
	})
	_ = file.SetCellStyle(helpSheet, "A1", "A1", titleStyle)

	_ = file.SetDocProps(&excelize.DocProperties{
		Title:       "ResumeTrail Import Template",
		Subject:     "Resume application tracker",
		Creator:     "ResumeTrail",
		Description: "Excel import template for resume applications",
	})

	file.SetActiveSheet(0)
	return file, nil
}

func (s *Server) importApplications(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxExcelSize+1<<20)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		apiError(c, http.StatusBadRequest, "file_required", "Please choose an Excel file.")
		return
	}
	if fileHeader.Size > maxExcelSize {
		apiError(c, http.StatusRequestEntityTooLarge, "file_too_large", "The Excel file is larger than 10 MB.")
		return
	}
	ext := strings.ToLower(fileHeader.Filename)
	if !strings.HasSuffix(ext, ".xlsx") && !strings.HasSuffix(ext, ".xlsm") {
		apiError(c, http.StatusBadRequest, "unsupported_file", "Only .xlsx and .xlsm files are supported.")
		return
	}

	uploaded, err := fileHeader.Open()
	if err != nil {
		apiError(c, http.StatusBadRequest, "file_open_failed", "Failed to read the uploaded file.")
		return
	}
	defer uploaded.Close()

	file, err := excelize.OpenReader(io.LimitReader(uploaded, maxExcelSize))
	if err != nil {
		apiError(c, http.StatusBadRequest, "invalid_excel", "The file is not a valid Excel workbook.")
		return
	}
	defer func() {
		_ = file.Close()
	}()

	result, err := s.parseAndImportWorkbook(file)
	if err != nil {
		apiError(c, http.StatusBadRequest, "import_failed", err.Error())
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) parseAndImportWorkbook(file *excelize.File) (*importResult, error) {
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("the workbook has no sheets")
	}
	rows, err := file.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("read the first sheet: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("the first sheet is empty")
	}

	headerRow, columns := findImportHeader(rows)
	if headerRow < 0 {
		return nil, fmt.Errorf("could not find a valid header row (company name column is required)")
	}
	if len(rows)-headerRow-1 > maxExcelRows {
		return nil, fmt.Errorf("the workbook contains more than %d data rows", maxExcelRows)
	}

	result := &importResult{Errors: []string{}}
	records := make([]models.Application, 0, len(rows))
	appliedAt := time.Now()
	for index := headerRow + 1; index < len(rows); index++ {
		row := rows[index]
		rowNumber := index + 1
		if rowIsEmpty(row) {
			continue
		}

		company := strings.TrimSpace(cellAt(row, columns.Company))
		if company == "" {
			result.Skipped++
			addImportError(result, rowNumber, "company name is required")
			continue
		}
		if utf8.RuneCountInString(company) > 255 {
			result.Skipped++
			addImportError(result, rowNumber, "company name is too long")
			continue
		}

		category := models.CategorySpring
		if raw := strings.TrimSpace(cellAt(row, columns.Category)); raw != "" {
			value, ok := parseCategory(raw)
			if !ok {
				result.Skipped++
				addImportError(result, rowNumber, "unknown category: "+raw)
				continue
			}
			category = value
		}

		status := models.StatusScreening
		if raw := strings.TrimSpace(cellAt(row, columns.Status)); raw != "" {
			value, ok := parseStatus(raw)
			if !ok {
				result.Skipped++
				addImportError(result, rowNumber, "unknown status: "+raw)
				continue
			}
			status = value
		}

		link := normalizeLink(cellAt(row, columns.Link))
		if len(link) > 2048 {
			result.Skipped++
			addImportError(result, rowNumber, "application link is too long")
			continue
		}

		records = append(records, models.Application{
			Category:    category,
			CompanyName: company,
			Link:        link,
			Status:      status,
			AppliedAt:   appliedAt,
		})
	}

	if len(records) > 0 {
		if err := s.database().CreateInBatches(&records, 200).Error; err != nil {
			return nil, fmt.Errorf("save imported applications: %w", err)
		}
	}
	result.Imported = len(records)
	result.Total = result.Imported + result.Skipped
	return result, nil
}

func findImportHeader(rows [][]string) (int, importColumns) {
	columns := importColumns{Category: -1, Company: -1, Link: -1, Status: -1}
	limit := len(rows)
	if limit > 10 {
		limit = 10
	}
	for rowIndex := 0; rowIndex < limit; rowIndex++ {
		columns = importColumns{Category: -1, Company: -1, Link: -1, Status: -1}
		for colIndex, cell := range rows[rowIndex] {
			switch {
			case columns.Company < 0 && matchesHeader(cell, []string{"company", "companyname", "企业名称", "公司名称", "企业", "公司", "会社名", "기업명", "회사명"}):
				columns.Company = colIndex
			case columns.Category < 0 && matchesHeader(cell, []string{"category", "大分类", "分类", "类别", "招聘类型", "招聘類型", "採用区分", "채용유형"}):
				columns.Category = colIndex
			case columns.Link < 0 && matchesHeader(cell, []string{"link", "url", "applicationlink", "投递链接", "投遞連結", "链接", "連結", "网址", "網址", "応募リンク", "지원링크"}):
				columns.Link = colIndex
			case columns.Status < 0 && matchesHeader(cell, []string{"status", "状态", "狀態", "进度", "進度", "進捗", "진행상태"}):
				columns.Status = colIndex
			}
		}
		if columns.Company >= 0 {
			return rowIndex, columns
		}
	}
	return -1, columns
}

func matchesHeader(value string, aliases []string) bool {
	normalized := normalizeHeader(value)
	if normalized == "" {
		return false
	}
	for _, alias := range aliases {
		if strings.Contains(normalized, normalizeHeader(alias)) {
			return true
		}
	}
	return false
}

func normalizeHeader(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		" ", "", "\t", "", "\n", "", "\r", "",
		"/", "", "\\", "", "-", "", "_", "",
		"(", "", ")", "", "（", "", "）", "",
		"、", "", "|", "", ":", "", "：", "",
	)
	return replacer.Replace(lower)
}

func parseCategory(value string) (string, bool) {
	key := normalizeHeader(value)
	if category, ok := categoryAliases[key]; ok {
		return category, true
	}
	// Accept values such as "Spring 2026".
	for alias, category := range categoryAliases {
		if strings.HasPrefix(key, alias) {
			return category, true
		}
	}
	return "", false
}

func parseStatus(value string) (string, bool) {
	key := normalizeHeader(value)
	if status, ok := statusAliases[key]; ok {
		return status, true
	}
	for alias, status := range statusAliases {
		if strings.HasPrefix(key, alias) {
			return status, true
		}
	}
	return "", false
}

func cellAt(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func rowIsEmpty(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func addImportError(result *importResult, row int, message string) {
	if len(result.Errors) >= 20 {
		return
	}
	result.Errors = append(result.Errors, "Row "+strconv.Itoa(row)+": "+message)
}

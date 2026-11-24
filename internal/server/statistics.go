package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"resume-helper/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// parseStatsRange reads the optional from/to query parameters used by the
// statistics page. An absent value means "no bound"; a malformed value is
// reported to the client instead of silently returning the whole table.
func parseStatsRange(c *gin.Context) (*time.Time, *time.Time, bool) {
	parse := func(key string) (*time.Time, bool) {
		raw := strings.TrimSpace(c.Query(key))
		if raw == "" {
			return nil, true
		}
		value, err := parseFlexibleTime(raw)
		if err != nil {
			apiError(c, http.StatusBadRequest, "invalid_range", "Invalid statistics date range.")
			return nil, false
		}
		return &value, true
	}

	from, ok := parse("from")
	if !ok {
		return nil, nil, false
	}
	to, ok := parse("to")
	if !ok {
		return nil, nil, false
	}
	if from != nil && to != nil && to.Before(*from) {
		apiError(c, http.StatusBadRequest, "invalid_range", "The end date must be after the start date.")
		return nil, nil, false
	}
	return from, to, true
}

// parseStatsCategory reads the optional category filter used by the
// statistics page. "all" (and an absent value) means no filter.
func parseStatsCategory(c *gin.Context) (string, bool) {
	raw := strings.TrimSpace(c.Query("category"))
	if raw == "" || strings.EqualFold(raw, "all") {
		return "", true
	}
	category := models.NormalizeCategory(raw)
	if !models.IsValidCategory(category) {
		apiError(c, http.StatusBadRequest, "invalid_category", "Unknown application category.")
		return "", false
	}
	return category, true
}

func (s *Server) applicationsInRange(db *gorm.DB, from, to *time.Time, category string) ([]models.Application, error) {
	query := db.Model(&models.Application{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if from != nil {
		query = query.Where("applied_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("applied_at <= ?", *to)
	}
	var items []models.Application
	err := query.Order("applied_at ASC, id ASC").Find(&items).Error
	return items, err
}

type funnelStage struct {
	Key        string  `json:"key"`
	Count      int64   `json:"count"`
	Conversion float64 `json:"conversion"`
}

type companyFunnel struct {
	Company              string  `json:"company"`
	Total                int64   `json:"total"`
	Interviews           int64   `json:"interviews"`
	Offers               int64   `json:"offers"`
	InterviewRate        float64 `json:"interviewRate"`
	InterviewToOfferRate float64 `json:"interviewToOfferRate"`
	OfferRate            float64 `json:"offerRate"`
}

type funnelResponse struct {
	Total                   int64           `json:"total"`
	Stages                  []funnelStage   `json:"stages"`
	AverageDaysToInterview  float64         `json:"averageDaysToInterview"`
	AverageInterviewToOffer float64         `json:"averageInterviewToOffer"`
	Companies               []companyFunnel `json:"companies"`
	InterviewCount          int64           `json:"interviewCount"`
	OfferCount              int64           `json:"offerCount"`
}

func (s *Server) applicationFunnel(c *gin.Context) {
	from, to, ok := parseStatsRange(c)
	if !ok {
		return
	}
	category, ok := parseStatsCategory(c)
	if !ok {
		return
	}

	db := s.database()
	applications, err := s.applicationsInRange(db, from, to, category)
	if err != nil {
		apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load the conversion funnel.")
		return
	}

	ids := make([]uint, 0, len(applications))
	for _, application := range applications {
		ids = append(ids, application.ID)
	}

	changesByApplication := make(map[uint][]models.ApplicationChange, len(ids))
	if len(ids) > 0 {
		var changes []models.ApplicationChange
		if err := db.Where("application_id IN ?", ids).
			Order("changed_at ASC, id ASC").
			Find(&changes).Error; err != nil {
			apiError(c, http.StatusInternalServerError, "query_failed", "Failed to load the conversion funnel.")
			return
		}
		for _, change := range changes {
			changesByApplication[change.ApplicationID] = append(changesByApplication[change.ApplicationID], change)
		}
	}

	reached := map[string]int64{}
	companyStats := map[string]*companyFunnel{}
	var interviewDaysTotal float64
	var interviewDaysCount int64
	var offerIntervalTotal float64
	var offerIntervalCount int64

	for _, application := range applications {
		// Use the same authoritative timeline as the detail page so milestones
		// that were rolled back no longer count toward later funnel stages.
		changes := reconcileApplicationChanges(application, changesByApplication[application.ID])
		statuses := map[string]bool{}
		for _, change := range changes {
			statuses[change.Status] = true
		}
		for _, status := range models.StageChainForStatus(application.Status) {
			statuses[status] = true
		}

		for _, stage := range models.StageOrder {
			if statuses[stage] {
				reached[stage]++
			}
		}

		first := func(status string) (time.Time, bool) {
			for _, change := range changes {
				if change.Status == status {
					return change.ChangedAt, true
				}
			}
			return time.Time{}, false
		}

		if interviewAt, found := first(models.StatusInterview); found {
			days := interviewAt.Sub(application.AppliedAt).Hours() / 24
			if days >= 0 {
				interviewDaysTotal += days
				interviewDaysCount++
			}
			if offerAt, hasOffer := first(models.StatusOffer); hasOffer {
				interval := offerAt.Sub(interviewAt).Hours() / 24
				if interval >= 0 {
					offerIntervalTotal += interval
					offerIntervalCount++
				}
			}
		}

		company := strings.TrimSpace(application.CompanyName)
		if company == "" {
			company = "—"
		}
		stat, exists := companyStats[company]
		if !exists {
			stat = &companyFunnel{Company: company}
			companyStats[company] = stat
		}
		stat.Total++
		if statuses[models.StatusInterview] || statuses[models.StatusOffer] {
			stat.Interviews++
		}
		if statuses[models.StatusOffer] {
			stat.Offers++
		}
	}

	total := int64(len(applications))
	stages := make([]funnelStage, 0, len(models.StageOrder)+1)
	stages = append(stages, funnelStage{Key: "applied", Count: total, Conversion: percent(total, total)})
	for _, stage := range models.StageOrder {
		count := reached[stage]
		stages = append(stages, funnelStage{
			Key:        stage,
			Count:      count,
			Conversion: percent(count, total),
		})
	}

	companies := make([]companyFunnel, 0, len(companyStats))
	for _, stat := range companyStats {
		stat.InterviewRate = percent(stat.Interviews, stat.Total)
		stat.OfferRate = percent(stat.Offers, stat.Total)
		stat.InterviewToOfferRate = percent(stat.Offers, stat.Interviews)
		companies = append(companies, *stat)
	}
	sort.SliceStable(companies, func(left, right int) bool {
		if companies[left].Total == companies[right].Total {
			return companies[left].Company < companies[right].Company
		}
		return companies[left].Total > companies[right].Total
	})
	if len(companies) > 12 {
		companies = companies[:12]
	}

	response := funnelResponse{
		Total:          total,
		Stages:         stages,
		Companies:      companies,
		InterviewCount: reached[models.StatusInterview],
		OfferCount:     reached[models.StatusOffer],
	}
	if interviewDaysCount > 0 {
		response.AverageDaysToInterview = round1(interviewDaysTotal / float64(interviewDaysCount))
	}
	if offerIntervalCount > 0 {
		response.AverageInterviewToOffer = round1(offerIntervalTotal / float64(offerIntervalCount))
	}

	c.JSON(http.StatusOK, response)
}

func percent(part, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return round1(float64(part) / float64(total) * 100)
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

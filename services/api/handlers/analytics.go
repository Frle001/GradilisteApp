package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/services"
)

type AnalyticsHandler struct {
	svc *services.AnalyticsService
}

func NewAnalyticsHandler(svc *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{svc: svc}
}

// ── Error helper ──────────────────────────────────────────────────────────────

func respondAnalyticsError(c *gin.Context, err error) {
	var ve *services.ValidationError
	switch {
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Pristup nije dozvoljen"})
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Zaposlenik nije pronađen"})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Interna greška"})
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func (h *AnalyticsHandler) GetMonthlyLaborSummary(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	year, month := parseYearMonth(c)

	summary, err := h.svc.GetMonthlyLaborSummary(c.Request.Context(), u.CompanyID, u.Role, year, month)
	if err != nil {
		respondAnalyticsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

func (h *AnalyticsHandler) GetEmployeeLaborCost(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	year, month := parseYearMonth(c)
	employeeID := c.Param("employeeId")

	cost, err := h.svc.GetEmployeeLaborCost(c.Request.Context(), u.CompanyID, u.Role, employeeID, year, month)
	if err != nil {
		respondAnalyticsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"detail": cost})
}

func (h *AnalyticsHandler) ListCompensationPlans(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	employeeID := c.Param("employeeId")

	plans, err := h.svc.ListCompensationPlans(c.Request.Context(), u.CompanyID, u.Role, employeeID)
	if err != nil {
		respondAnalyticsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

func (h *AnalyticsHandler) CreateCompensationPlan(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	employeeID := c.Param("employeeId")

	var req dto.CreateCompensationPlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}

	plan, err := h.svc.CreateCompensationPlan(c.Request.Context(), u.CompanyID, u.Role, u.UserID, employeeID, req)
	if err != nil {
		respondAnalyticsError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"plan": plan})
}

func (h *AnalyticsHandler) UpdateCompensationPlan(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	planID := c.Param("planId")

	var req dto.UpdateCompensationPlanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}

	plan, err := h.svc.UpdateCompensationPlan(c.Request.Context(), u.CompanyID, u.Role, planID, req)
	if err != nil {
		respondAnalyticsError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"plan": plan})
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func parseYearMonth(c *gin.Context) (int, int) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if y, err := strconv.Atoi(c.Query("year")); err == nil && y >= 2000 && y <= 2100 {
		year = y
	}
	if m, err := strconv.Atoi(c.Query("month")); err == nil && m >= 1 && m <= 12 {
		month = m
	}
	return year, month
}

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

type RevenueHandler struct {
	svc *services.RevenueService
}

func NewRevenueHandler(svc *services.RevenueService) *RevenueHandler {
	return &RevenueHandler{svc: svc}
}

func respondRevenueError(c *gin.Context, err error) {
	var ve *services.ValidationError
	switch {
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Pristup nije dozvoljen"})
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Nije pronađeno"})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Interna greška"})
	}
}

func parseYear(c *gin.Context) int {
	if y, err := strconv.Atoi(c.Query("year")); err == nil && y >= 2000 && y <= 2100 {
		return y
	}
	return time.Now().Year()
}

func parseMonth(c *gin.Context) int {
	if m, err := strconv.Atoi(c.Query("month")); err == nil && m >= 1 && m <= 12 {
		return m
	}
	return 0
}

// ── Analytics ─────────────────────────────────────────────────────────────────

func (h *RevenueHandler) GetRevenueSummary(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	year := parseYear(c)
	month := parseMonth(c)

	summary, err := h.svc.GetRevenueSummary(c.Request.Context(), u.CompanyID, u.Role, year, month)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

func (h *RevenueHandler) GetProjectRevenueDetail(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Param("projectId")
	year := parseYear(c)

	detail, err := h.svc.GetProjectRevenueDetail(c.Request.Context(), u.CompanyID, u.Role, projectID, year)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"detail": detail})
}

// ── Invoices ──────────────────────────────────────────────────────────────────

func (h *RevenueHandler) ListInvoices(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var projectID, status *string
	if p := c.Query("project_id"); p != "" {
		projectID = &p
	}
	if s := c.Query("status"); s != "" {
		status = &s
	}
	year := 0
	if y, err := strconv.Atoi(c.Query("year")); err == nil && y >= 2000 && y <= 2100 {
		year = y
	}
	month := 0
	if m, err := strconv.Atoi(c.Query("month")); err == nil && m >= 1 && m <= 12 {
		month = m
	}

	invoices, err := h.svc.ListInvoices(c.Request.Context(), u.CompanyID, u.Role, projectID, status, year, month)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}

func (h *RevenueHandler) CreateInvoice(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.CreateProjectInvoiceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}

	inv, err := h.svc.CreateInvoice(c.Request.Context(), u.CompanyID, u.Role, u.UserID, req)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"invoice": inv})
}

func (h *RevenueHandler) GetInvoice(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	invoiceID := c.Param("id")

	inv, err := h.svc.GetInvoice(c.Request.Context(), u.CompanyID, u.Role, invoiceID)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"invoice": inv})
}

func (h *RevenueHandler) CancelInvoice(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	invoiceID := c.Param("id")

	var req dto.CancelProjectInvoiceReq
	_ = c.ShouldBindJSON(&req) // notes optional

	if err := h.svc.CancelInvoice(c.Request.Context(), u.CompanyID, u.Role, invoiceID, req.Notes); err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *RevenueHandler) AddPayment(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	invoiceID := c.Param("id")

	var req dto.AddPaymentReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}

	pay, err := h.svc.AddPayment(c.Request.Context(), u.CompanyID, u.Role, u.UserID, invoiceID, req)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"payment": pay})
}

func (h *RevenueHandler) ListPayments(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	invoiceID := c.Param("id")

	payments, err := h.svc.ListPayments(c.Request.Context(), u.CompanyID, u.Role, invoiceID)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"payments": payments})
}

// ── Contracts ─────────────────────────────────────────────────────────────────

func (h *RevenueHandler) ListContracts(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Query("project_id")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id je obavezan"})
		return
	}

	contracts, err := h.svc.ListContractsForProject(c.Request.Context(), u.CompanyID, u.Role, projectID)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"contracts": contracts})
}

func (h *RevenueHandler) CreateContract(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var req dto.CreateContractReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}

	contract, err := h.svc.CreateContract(c.Request.Context(), u.CompanyID, u.Role, u.UserID, req)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"contract": contract})
}

func (h *RevenueHandler) GetContract(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	contractID := c.Param("id")

	contract, err := h.svc.GetContract(c.Request.Context(), u.CompanyID, u.Role, contractID)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"contract": contract})
}

func (h *RevenueHandler) UpdateContract(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	contractID := c.Param("id")

	var req dto.UpdateContractReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev"})
		return
	}

	contract, err := h.svc.UpdateContract(c.Request.Context(), u.CompanyID, u.Role, contractID, req)
	if err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"contract": contract})
}

func (h *RevenueHandler) DeleteContract(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	contractID := c.Param("id")

	if err := h.svc.DeleteContract(c.Request.Context(), u.CompanyID, u.Role, contractID); err != nil {
		respondRevenueError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

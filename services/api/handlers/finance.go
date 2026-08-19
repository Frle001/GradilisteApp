package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/services"
)

type FinanceHandler struct {
	svc *services.FinanceService
}

func NewFinanceHandler(svc *services.FinanceService) *FinanceHandler {
	return &FinanceHandler{svc: svc}
}

// ── Računi ────────────────────────────────────────────────────────────────────

// POST /api/finance/racuni
func (h *FinanceHandler) CreateInvoice(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dokument je obavezan"})
		return
	}
	defer file.Close()

	invoiceType := c.PostForm("invoice_type")
	var supplier *string
	if s := c.PostForm("supplier"); s != "" {
		supplier = &s
	}
	var leasingCompany *string
	if lc := c.PostForm("leasing_company"); lc != "" {
		leasingCompany = &lc
	}

	inv, err := h.svc.CreateInvoice(
		c.Request.Context(),
		u.CompanyID, u.Role, u.UserID,
		invoiceType, supplier, leasingCompany,
		file, header,
	)
	if err != nil {
		respondFinanceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"invoice": inv})
}

// GET /api/finance/racuni
func (h *FinanceHandler) ListInvoices(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	list, err := h.svc.ListInvoices(c.Request.Context(), u.CompanyID, u.Role)
	if err != nil {
		respondFinanceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"invoices": list})
}

// GET /api/finance/racuni/:invoiceId/download
func (h *FinanceHandler) DownloadInvoice(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	reader, ct, filename, err := h.svc.DownloadInvoice(
		c.Request.Context(),
		u.CompanyID, u.Role, c.Param("invoiceId"),
	)
	if err != nil {
		respondFinanceError(c, err)
		return
	}
	defer reader.Close()
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.DataFromReader(http.StatusOK, -1, ct, reader, nil)
}

// ── R1 Receipts ───────────────────────────────────────────────────────────────

// POST /api/finance/r1
func (h *FinanceHandler) CreateR1Receipt(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dokument je obavezan"})
		return
	}
	defer file.Close()

	var body struct {
		Price float64 `form:"price"`
	}
	if err := c.ShouldBind(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rec, err := h.svc.CreateR1Receipt(
		c.Request.Context(),
		u.CompanyID, u.Role, u.UserID,
		body.Price,
		file, header,
	)
	if err != nil {
		respondFinanceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"receipt": rec})
}

// GET /api/finance/r1
func (h *FinanceHandler) ListR1Receipts(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	list, err := h.svc.ListR1Receipts(c.Request.Context(), u.CompanyID, u.Role, u.UserID)
	if err != nil {
		respondFinanceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"receipts": list})
}

// GET /api/finance/r1/:receiptId/download
func (h *FinanceHandler) DownloadR1Receipt(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	reader, ct, filename, err := h.svc.DownloadR1Receipt(
		c.Request.Context(),
		u.CompanyID, u.Role, u.UserID, c.Param("receiptId"),
	)
	if err != nil {
		respondFinanceError(c, err)
		return
	}
	defer reader.Close()
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.DataFromReader(http.StatusOK, -1, ct, reader, nil)
}

// ── Error helper ──────────────────────────────────────────────────────────────

func respondFinanceError(c *gin.Context, err error) {
	var ve *services.ValidationError
	switch {
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	default:
		log.Printf("[finance] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}

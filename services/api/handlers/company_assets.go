package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/services"
)

type CompanyAssetsHandler struct {
	svc *services.CompanyAssetsService
}

func NewCompanyAssetsHandler(svc *services.CompanyAssetsService) *CompanyAssetsHandler {
	return &CompanyAssetsHandler{svc: svc}
}

// List GET /api/company-assets
func (h *CompanyAssetsHandler) List(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	f := dto.CompanyAssetFilter{
		AssetType:  c.Query("type"),
		EmployeeID: c.Query("employee_id"),
		Status:     c.Query("status"),
		Search:     c.Query("search"),
	}

	assets, err := h.svc.List(c.Request.Context(), u.CompanyID, u.Role, f)
	if err != nil {
		respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"assets": assets, "total": len(assets)})
}

// GetByID GET /api/company-assets/:id
func (h *CompanyAssetsHandler) GetByID(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	asset, err := h.svc.GetByID(c.Request.Context(), u.CompanyID, u.Role, id)
	if err != nil {
		respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"asset": asset})
}

// Create POST /api/company-assets
func (h *CompanyAssetsHandler) Create(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	var req dto.CreateCompanyAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	asset, err := h.svc.Create(c.Request.Context(), u.CompanyID, u.Role, req)
	if err != nil {
		respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"asset": asset})
}

// Update PATCH /api/company-assets/:id
func (h *CompanyAssetsHandler) Update(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	var req dto.UpdateCompanyAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	asset, err := h.svc.Update(c.Request.Context(), u.CompanyID, u.Role, id, req)
	if err != nil {
		respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"asset": asset})
}

// Deactivate DELETE /api/company-assets/:id
func (h *CompanyAssetsHandler) Deactivate(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	if err := h.svc.Deactivate(c.Request.Context(), u.CompanyID, u.Role, id); err != nil {
		respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset deactivated"})
}

// FormEmployees GET /api/company-assets/employees
func (h *CompanyAssetsHandler) FormEmployees(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	employees, err := h.svc.ListEmployees(c.Request.Context(), u.CompanyID, u.Role)
	if err != nil {
		respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"employees": employees})
}

// Notifications GET /api/company-assets/notifications
func (h *CompanyAssetsHandler) Notifications(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	notes, err := h.svc.GetNotifications(c.Request.Context(), u.CompanyID, u.Role, time.Now().UTC())
	if err != nil {
		respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, dto.NotificationsResponse{
		Notifications: notes,
		Count:         len(notes),
	})
}

// MarkLeasingPayment POST /api/company-assets/:id/leasing-payments
func (h *CompanyAssetsHandler) MarkLeasingPayment(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	var req dto.LeasingPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	payment, err := h.svc.MarkLeasingPayment(c.Request.Context(), u.CompanyID, u.Role, u.UserID, id, req)
	if err != nil {
		respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"payment": payment})
}

// GetLeasingHistory GET /api/company-assets/:id/leasing-payments
func (h *CompanyAssetsHandler) GetLeasingHistory(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	history, err := h.svc.GetLeasingHistory(c.Request.Context(), u.CompanyID, u.Role, id)
	if err != nil {
		respondAssetError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"payments": history, "total": len(history)})
}

// ── Error mapping ─────────────────────────────────────────────────────────────

func respondAssetError(c *gin.Context, err error) {
	var ve *services.ValidationError
	switch {
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	default:
		log.Printf("[company_assets] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}

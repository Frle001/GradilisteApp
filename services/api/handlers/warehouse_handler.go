package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/services"
)

type WarehouseHandler struct {
	svc *services.WarehouseService
}

func NewWarehouseHandler(svc *services.WarehouseService) *WarehouseHandler {
	return &WarehouseHandler{svc: svc}
}

// GET /api/warehouse/overview
func (h *WarehouseHandler) GetOverview(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	resp, err := h.svc.GetOverview(c.Request.Context(), u.CompanyID, u.Role)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/warehouse/stock
func (h *WarehouseHandler) ListStock(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	search := c.Query("search")
	activeOnly := c.Query("active_only") != "false"

	items, err := h.svc.ListStock(c.Request.Context(), u.CompanyID, u.Role, search, activeOnly)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// GET /api/warehouse/stock/:balanceId
func (h *WarehouseHandler) GetStock(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	balanceID := c.Param("balanceId")
	detail, err := h.svc.GetStockByID(c.Request.Context(), u.CompanyID, u.Role, balanceID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, detail)
}

// POST /api/warehouse/receipts
func (h *WarehouseHandler) CreateReceipt(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	var req dto.CreateReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	idempotencyKey := c.GetHeader("X-Idempotency-Key")
	resp, isReplay, err := h.svc.CreateReceipt(c.Request.Context(), u.CompanyID, u.Role, u.UserID, idempotencyKey, req)
	if err != nil {
		h.respondError(c, err)
		return
	}
	status := http.StatusCreated
	if isReplay {
		status = http.StatusOK
	}
	c.JSON(status, resp)
}

// POST /api/warehouse/issues
func (h *WarehouseHandler) IssueToProject(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	var req dto.CreateIssueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	idempotencyKey := c.GetHeader("X-Idempotency-Key")
	movID, isReplay, err := h.svc.IssueToProject(c.Request.Context(), u.CompanyID, u.Role, u.UserID, u.EmployeeID, idempotencyKey, req)
	if err != nil {
		h.respondError(c, err)
		return
	}
	status := http.StatusCreated
	if isReplay {
		status = http.StatusOK
	}
	c.JSON(status, dto.MovementIDResponse{ID: movID})
}

// POST /api/warehouse/returns
func (h *WarehouseHandler) ReturnFromProject(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	var req dto.CreateReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	idempotencyKey := c.GetHeader("X-Idempotency-Key")
	movID, isReplay, err := h.svc.ReturnFromProject(c.Request.Context(), u.CompanyID, u.Role, u.UserID, u.EmployeeID, idempotencyKey, req)
	if err != nil {
		h.respondError(c, err)
		return
	}
	status := http.StatusCreated
	if isReplay {
		status = http.StatusOK
	}
	c.JSON(status, dto.MovementIDResponse{ID: movID})
}

// POST /api/warehouse/adjustments
func (h *WarehouseHandler) AdjustInventory(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	var req dto.CreateAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	idempotencyKey := c.GetHeader("X-Idempotency-Key")
	movID, isReplay, err := h.svc.AdjustInventory(c.Request.Context(), u.CompanyID, u.Role, u.UserID, idempotencyKey, req)
	if err != nil {
		h.respondError(c, err)
		return
	}
	status := http.StatusCreated
	if isReplay {
		status = http.StatusOK
	}
	c.JSON(status, dto.MovementIDResponse{ID: movID})
}

// GET /api/warehouse/movements
func (h *WarehouseHandler) ListMovements(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "25"))

	var movementType, balanceID *string
	if mt := c.Query("type"); mt != "" {
		movementType = &mt
	}
	if bid := c.Query("balance_id"); bid != "" {
		balanceID = &bid
	}

	resp, err := h.svc.ListMovements(c.Request.Context(), u.CompanyID, u.Role, movementType, balanceID, page, perPage)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *WarehouseHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrWarehouseAccessDenied):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrWarehouseNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrWarehouseInsufficientQty):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrIdempotencyConflict):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

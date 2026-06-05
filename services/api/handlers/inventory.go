package handlers

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/services"
)

type InventoryHandler struct {
	svc *services.InventoryService
}

func NewInventoryHandler(svc *services.InventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

// GET /api/inventory/me
func (h *InventoryHandler) GetMyInventory(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	if u.EmployeeID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "no employee record linked to your account"})
		return
	}

	resp, err := h.svc.GetMyInventory(c.Request.Context(), u.CompanyID, u.EmployeeID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/inventory/employees/:employeeId
func (h *InventoryHandler) GetEmployeeInventory(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	employeeID := c.Param("employeeId")
	resp, err := h.svc.GetEmployeeInventory(c.Request.Context(), u.CompanyID, u.Role, employeeID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/inventory/transfer-targets
func (h *InventoryHandler) GetTransferTargets(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	if u.EmployeeID == "" {
		c.JSON(http.StatusOK, gin.H{"targets": []struct{}{}})
		return
	}

	transferType := c.DefaultQuery("transfer_type", "asset")
	targets, err := h.svc.GetTransferTargets(c.Request.Context(), u.CompanyID, u.EmployeeID, u.Role, transferType)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"targets": targets})
}

// GET /api/inventory/employee-projects
func (h *InventoryHandler) GetEmployeeProjects(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	employeeID := c.Query("employee_id")
	if employeeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "employee_id is required"})
		return
	}
	projects, err := h.svc.GetEmployeeProjects(c.Request.Context(), u.CompanyID, u.EmployeeID, employeeID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

// POST /api/inventory/transfers
func (h *InventoryHandler) CreateTransfer(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	id, err := h.svc.CreateTransfer(c.Request.Context(), u.CompanyID, u.EmployeeID, u.Role, u.UserID, body)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// GET /api/inventory/transfers
func (h *InventoryHandler) ListTransfers(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "25"))

	resp, err := h.svc.ListTransfers(c.Request.Context(), u.CompanyID, u.EmployeeID, u.Role, page, perPage)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *InventoryHandler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInventoryEmployeeNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrAssetNotFound):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInsufficientMaterial):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrNoMaterialResponsibility):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrCannotTransferToSelf):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrTransferTargetNotAllowed):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrMaterialRecipientNotPoslovoda):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInvalidTransferType):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrMissingTransferFields):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInvalidTransferQuantity):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrInvalidDestinationProject):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		log.Printf("[inventory] unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

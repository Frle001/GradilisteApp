package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/services"
)

type EmployeeHandler struct {
	svc *services.EmployeeService
}

func NewEmployeeHandler(svc *services.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

// ── Employees ─────────────────────────────────────────────────────────────────

// List GET /api/employees
func (h *EmployeeHandler) List(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	filter := services.EmployeeFilter{
		Search: c.Query("search"),
	}
	if r := c.Query("role"); r != "" {
		filter.Role = &r
	}
	switch c.Query("active") {
	case "true":
		t := true
		filter.Active = &t
	case "false":
		f := false
		filter.Active = &f
	}

	employees, err := h.svc.List(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"employees": employees, "count": len(employees)})
}

// GetByID GET /api/employees/:id
func (h *EmployeeHandler) GetByID(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	emp, err := h.svc.GetByID(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, id)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"employee": emp})
}

// Create POST /api/employees
func (h *EmployeeHandler) Create(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	var req dto.CreateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	result, err := h.svc.Create(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	msg := "Employee created. No login account created for radnik role."
	if result.LoginCreated {
		msg = "Employee created. Login account created with temporary password."
	}

	c.JSON(http.StatusCreated, gin.H{
		"employee":           result.Employee,
		"loginCreated":       result.LoginCreated,
		"temporaryPassword":  result.TemporaryPassword,
		"mustChangePassword": result.MustChangePassword,
		"message":            msg,
	})
}

// Update PUT /api/employees/:id
func (h *EmployeeHandler) Update(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	var req dto.UpdateEmployeeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	emp, err := h.svc.Update(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), id, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"employee": emp})
}

// Deactivate PATCH /api/employees/:id/deactivate
func (h *EmployeeHandler) Deactivate(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	if err := h.svc.SetActive(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), id, false); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee deactivated"})
}

// Activate PATCH /api/employees/:id/activate
func (h *EmployeeHandler) Activate(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	if err := h.svc.SetActive(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), id, true); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Employee activated"})
}

// ── Assets ────────────────────────────────────────────────────────────────────

// ListAssets GET /api/employees/:id/assets
func (h *EmployeeHandler) ListAssets(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	employeeID := c.Param("id")

	assets, err := h.svc.ListAssets(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, employeeID)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"assets": assets, "count": len(assets)})
}

// CreateAsset POST /api/employees/:id/assets
func (h *EmployeeHandler) CreateAsset(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	employeeID := c.Param("id")

	var req dto.CreateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	asset, err := h.svc.CreateAsset(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), employeeID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"asset": asset})
}

// UpdateAsset PUT /api/employees/:id/assets/:asset_id
func (h *EmployeeHandler) UpdateAsset(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	employeeID := c.Param("id")
	assetID := c.Param("asset_id")

	var req dto.UpdateAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	asset, err := h.svc.UpdateAsset(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), employeeID, assetID, req)
	if err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"asset": asset})
}

// DeactivateAsset PATCH /api/employees/:id/assets/:asset_id/deactivate
func (h *EmployeeHandler) DeactivateAsset(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	employeeID := c.Param("id")
	assetID := c.Param("asset_id")

	if err := h.svc.DeactivateAsset(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), employeeID, assetID); err != nil {
		respondServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Asset deactivated"})
}

// ── Error helper ──────────────────────────────────────────────────────────────

func respondServiceError(c *gin.Context, err error) {
	var ve *services.ValidationError
	switch {
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	case errors.Is(err, services.ErrEmailInUse):
		c.JSON(http.StatusConflict, gin.H{"error": "Email address is already in use"})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}

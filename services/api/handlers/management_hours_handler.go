package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
	"github.com/gradiliste/api/services"
)

type ManagementHoursHandler struct {
	svc *services.ManagementHoursService
}

func NewManagementHoursHandler(svc *services.ManagementHoursService) *ManagementHoursHandler {
	return &ManagementHoursHandler{svc: svc}
}

// POST /api/management-hours
func (h *ManagementHoursHandler) Submit(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	var req dto.SubmitManagementHoursRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev: " + err.Error()})
		return
	}

	entry, err := h.svc.Submit(c.Request.Context(), u.CompanyID, u.EmployeeID, u.UserID, u.Role, req)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovoj operaciji"})
			return
		}
		if ve := services.AsValidationError(err); ve != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

// GET /api/management-hours
func (h *ManagementHoursHandler) List(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	entries, err := h.svc.List(c.Request.Context(), u.CompanyID, u.EmployeeID, u.Role)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovoj operaciji"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

// GET /api/management-hours/:id
func (h *ManagementHoursHandler) GetByID(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	entryID := c.Param("id")

	entry, err := h.svc.GetByID(c.Request.Context(), u.CompanyID, u.EmployeeID, u.Role, entryID)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovoj operaciji"})
			return
		}
		if errors.Is(err, repositories.ErrManagementHoursNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Unos nije pronađen"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

// PUT /api/management-hours/:id
func (h *ManagementHoursHandler) Update(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	entryID := c.Param("id")

	var req dto.UpdateManagementHoursRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev: " + err.Error()})
		return
	}

	if err := h.svc.Update(c.Request.Context(), u.CompanyID, u.EmployeeID, u.Role, entryID, req); err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovoj operaciji"})
			return
		}
		if errors.Is(err, repositories.ErrManagementHoursNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Unos nije pronađen"})
			return
		}
		if ve := services.AsValidationError(err); ve != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /api/management-hours/:id
func (h *ManagementHoursHandler) Delete(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	entryID := c.Param("id")

	if err := h.svc.Delete(c.Request.Context(), u.CompanyID, u.EmployeeID, u.Role, entryID); err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovoj operaciji"})
			return
		}
		if errors.Is(err, repositories.ErrManagementHoursNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Unos nije pronađen"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

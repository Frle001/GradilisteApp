package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/services"
)

type WorkerHoursHandler struct {
	svc *services.WorkerHoursService
}

func NewWorkerHoursHandler(svc *services.WorkerHoursService) *WorkerHoursHandler {
	return &WorkerHoursHandler{svc: svc}
}

// GET /api/worker-hours/projects
func (h *WorkerHoursHandler) ListProjects(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projects, err := h.svc.ListCompanyProjects(c.Request.Context(), u.CompanyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

// GET /api/worker-hours/my?date=YYYY-MM-DD
func (h *WorkerHoursHandler) ListMy(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	date := c.Query("date") // empty = today

	entries, err := h.svc.ListForDate(c.Request.Context(), u.CompanyID, u.EmployeeID, date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

// POST /api/worker-hours
func (h *WorkerHoursHandler) Submit(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	var req dto.SubmitWorkerHoursRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev: " + err.Error()})
		return
	}

	entry, err := h.svc.Submit(c.Request.Context(), u.CompanyID, u.EmployeeID, u.UserID, req)
	if err != nil {
		var ve *services.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
			return
		}
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Niste dodijeljeni na ovaj projekt"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

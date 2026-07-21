package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/services"
)

type ScheduleHandler struct {
	svc *services.ScheduleService
}

func NewScheduleHandler(svc *services.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{svc: svc}
}

// GET /api/schedule/shifts?date_from=YYYY-MM-DD&date_to=YYYY-MM-DD&project_id=xxx
func (h *ScheduleHandler) ListShifts(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")
	if dateFrom == "" || dateTo == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date_from i date_to su obavezni"})
		return
	}

	var projectID *string
	if pid := c.Query("project_id"); pid != "" {
		projectID = &pid
	}

	shifts, err := h.svc.ListShifts(c.Request.Context(), u.CompanyID, dateFrom, dateTo, projectID)
	if err != nil {
		if ve := services.AsValidationError(err); ve != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Error()})
			return
		}
		log.Printf("[schedule] ListShifts error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri dohvatu rasporeda"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"shifts": shifts})
}

// POST /api/schedule/shifts
func (h *ScheduleHandler) CreateShift(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var req dto.CreateShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "neispravni podaci: " + err.Error()})
		return
	}

	shift, err := h.svc.CreateShift(c.Request.Context(), u.CompanyID, req.ProjectID, u.UserID, &req)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "projekt nije pronađen"})
			return
		}
		if ve := services.AsValidationError(err); ve != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Error()})
			return
		}
		log.Printf("[schedule] CreateShift error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri kreiranju smjene"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"shift": shift})
}

// GET /api/schedule/shifts/:shiftId
func (h *ScheduleHandler) GetShift(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	shift, err := h.svc.GetShift(c.Request.Context(), u.CompanyID, c.Param("shiftId"))
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "smjena nije pronađena"})
			return
		}
		log.Printf("[schedule] GetShift error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri dohvatu smjene"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"shift": shift})
}

// PATCH /api/schedule/shifts/:shiftId
func (h *ScheduleHandler) UpdateShift(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var req dto.UpdateShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "neispravni podaci: " + err.Error()})
		return
	}

	shift, err := h.svc.UpdateShift(c.Request.Context(), u.CompanyID, c.Param("shiftId"), &req)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "smjena nije pronađena"})
			return
		}
		if errors.Is(err, services.ErrShiftCancelled) {
			c.JSON(http.StatusConflict, gin.H{"error": "smjena je otkazana"})
			return
		}
		if ve := services.AsValidationError(err); ve != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Error()})
			return
		}
		log.Printf("[schedule] UpdateShift error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri ažuriranju smjene"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"shift": shift})
}

// POST /api/schedule/shifts/:shiftId/cancel
func (h *ScheduleHandler) CancelShift(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	err := h.svc.CancelShift(c.Request.Context(), u.CompanyID, c.Param("shiftId"), u.UserID)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "smjena nije pronađena"})
			return
		}
		if errors.Is(err, services.ErrShiftCancelled) {
			c.JSON(http.StatusConflict, gin.H{"error": "smjena je već otkazana"})
			return
		}
		log.Printf("[schedule] CancelShift error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri otkazivanju smjene"})
		return
	}

	c.Status(http.StatusNoContent)
}

// PUT /api/schedule/shifts/:shiftId/assignments
func (h *ScheduleHandler) SyncAssignments(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var req dto.AssignEmployeesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "neispravni podaci: " + err.Error()})
		return
	}

	result, err := h.svc.SyncAssignments(c.Request.Context(), u.CompanyID, c.Param("shiftId"), u.UserID, &req)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "smjena nije pronađena"})
			return
		}
		if errors.Is(err, services.ErrShiftCancelled) {
			c.JSON(http.StatusConflict, gin.H{"error": "smjena je otkazana"})
			return
		}
		var overlapErr *services.ErrOverlapConflict
		if errors.As(err, &overlapErr) {
			c.JSON(http.StatusConflict, gin.H{
				"error":             "preklapanje rasporeda zaposlenika",
				"overlaps":          overlapErr.Overlaps,
				"requires_override": true,
			})
			return
		}
		log.Printf("[schedule] SyncAssignments error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri raspoređivanju"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// POST /api/schedule/shifts/:shiftId/duplicate
func (h *ScheduleHandler) DuplicateShift(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var req dto.DuplicateShiftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "neispravni podaci: " + err.Error()})
		return
	}

	shift, err := h.svc.DuplicateShift(c.Request.Context(), u.CompanyID, c.Param("shiftId"), u.UserID, &req)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "smjena nije pronađena"})
			return
		}
		if ve := services.AsValidationError(err); ve != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Error()})
			return
		}
		log.Printf("[schedule] DuplicateShift error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri dupliciranju smjene"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"shift": shift})
}

// POST /api/schedule/copy-day
func (h *ScheduleHandler) CopyDay(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var req dto.CopyDayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "neispravni podaci: " + err.Error()})
		return
	}

	result, err := h.svc.CopyDay(c.Request.Context(), u.CompanyID, u.UserID, &req)
	if err != nil {
		if ve := services.AsValidationError(err); ve != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Error()})
			return
		}
		log.Printf("[schedule] CopyDay error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri kopiranju dana"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// POST /api/schedule/copy-week
func (h *ScheduleHandler) CopyWeek(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var req dto.CopyWeekRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "neispravni podaci: " + err.Error()})
		return
	}

	result, err := h.svc.CopyWeek(c.Request.Context(), u.CompanyID, u.UserID, &req)
	if err != nil {
		if ve := services.AsValidationError(err); ve != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Error()})
			return
		}
		log.Printf("[schedule] CopyWeek error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri kopiranju tjedna"})
		return
	}

	c.JSON(http.StatusOK, result)
}

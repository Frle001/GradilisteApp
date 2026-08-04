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

type WorkerHoursHandler struct {
	svc *services.WorkerHoursService
}

func NewWorkerHoursHandler(svc *services.WorkerHoursService) *WorkerHoursHandler {
	return &WorkerHoursHandler{svc: svc}
}

// GET /api/worker-hours/projects
func (h *WorkerHoursHandler) ListProjects(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projects, err := h.svc.ListCompanyProjects(c.Request.Context(), u.CompanyID, u.Role)
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

// GET /api/worker-hours/history
func (h *WorkerHoursHandler) History(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	entries, err := h.svc.ListHistory(c.Request.Context(), u.CompanyID, u.EmployeeID)
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

	entry, err := h.svc.Submit(c.Request.Context(), u.CompanyID, u.EmployeeID, u.UserID, u.Role, req)
	if err != nil {
		var ve *services.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
			return
		}
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovoj operaciji"})
			return
		}
		if errors.Is(err, services.ErrSubmissionConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "Ovaj submission ID je već korišten s drugačijim podacima"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"entry": entry})
}

// ── Manager handlers ──────────────────────────────────────────────────────────

// GET /api/worker-hours/manager?project_id=&work_date=
func (h *WorkerHoursHandler) ManagerList(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Query("project_id")
	workDate := c.Query("work_date")

	entries, err := h.svc.ListManagerEntries(c.Request.Context(), u.CompanyID, u.Role, projectID, workDate)
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

// GET /api/worker-hours/manager/:id
func (h *WorkerHoursHandler) ManagerDetail(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	entryID := c.Param("id")

	detail, err := h.svc.GetDetail(c.Request.Context(), u.CompanyID, u.Role, entryID)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovoj operaciji"})
			return
		}
		if errors.Is(err, repositories.ErrWorkerHoursNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Unos nije pronađen"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entry": detail})
}

// POST /api/worker-hours/manager/:id/correct
func (h *WorkerHoursHandler) ManagerCorrect(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	entryID := c.Param("id")

	var req dto.CorrectWorkerHoursRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev: " + err.Error()})
		return
	}

	if err := h.svc.CorrectEntry(c.Request.Context(), u.CompanyID, u.Role, u.UserID, entryID, req); err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovoj operaciji"})
			return
		}
		var ve *services.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
			return
		}
		if errors.Is(err, repositories.ErrWorkerHoursNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Unos nije pronađen"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/worker-hours/manager/:id/comments
func (h *WorkerHoursHandler) ManagerAddComment(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	entryID := c.Param("id")

	var req dto.AddWorkerHoursCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Neispravan zahtjev: " + err.Error()})
		return
	}

	comment, err := h.svc.AddComment(c.Request.Context(), u.CompanyID, u.Role, u.UserID, entryID, req.CommentText)
	if err != nil {
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovoj operaciji"})
			return
		}
		var ve *services.ValidationError
		if errors.As(err, &ve) {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
			return
		}
		if errors.Is(err, repositories.ErrWorkerHoursNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Unos nije pronađen"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}

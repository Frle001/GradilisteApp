package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
	"github.com/gradiliste/api/services"
)

type ProjectHandler struct {
	svc *services.ProjectService
}

func NewProjectHandler(svc *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// ── List GET /api/projects ────────────────────────────────────────────────────

func (h *ProjectHandler) List(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	filter := services.ProjectFilter{
		Search: c.Query("search"),
	}
	if s := c.Query("status"); s != "" {
		filter.Status = &s
	}

	projects, err := h.svc.List(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"projects": projects, "count": len(projects)})
}

// ── GetByID GET /api/projects/:id ────────────────────────────────────────────

func (h *ProjectHandler) GetByID(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	project, err := h.svc.GetByID(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, id)
	if err != nil {
		respondProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"project": project})
}

// ── Create POST /api/projects ─────────────────────────────────────────────────

func (h *ProjectHandler) Create(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	project, err := h.svc.Create(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), req)
	if err != nil {
		respondProjectError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"project": project})
}

// ── Update PUT /api/projects/:id ──────────────────────────────────────────────

func (h *ProjectHandler) Update(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	var req dto.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	project, err := h.svc.Update(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), id, req)
	if err != nil {
		respondProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"project": project})
}

// ── AssignPoslovoda PATCH /api/projects/:id/assign-poslovoda ─────────────────

func (h *ProjectHandler) AssignPoslovoda(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	var req dto.AssignPoslovodaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	if err := h.svc.AssignPoslovoda(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), id, req.PoslovodaID); err != nil {
		respondProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Poslovođa assigned successfully"})
}

// ── Close PATCH /api/projects/:id/close ──────────────────────────────────────

func (h *ProjectHandler) Close(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	if err := h.svc.Close(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), id); err != nil {
		respondProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project closed"})
}

// ── Archive PATCH /api/projects/:id/archive ───────────────────────────────────

func (h *ProjectHandler) Archive(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	if err := h.svc.Archive(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), id); err != nil {
		respondProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project archived"})
}

// ── Reactivate PATCH /api/projects/:id/reactivate ────────────────────────────

func (h *ProjectHandler) Reactivate(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	if err := h.svc.Reactivate(c.Request.Context(), u.CompanyID, u.UserID, c.ClientIP(), c.GetHeader("User-Agent"), id); err != nil {
		respondProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Project reactivated"})
}

// ── Delete DELETE /api/projects/:id  (hard delete, direktor only) ────────────

func (h *ProjectHandler) Delete(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	if err := h.svc.HardDelete(c.Request.Context(), u.CompanyID, u.UserID, id); err != nil {
		respondProjectHardDeleteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ── ListAssignments GET /api/projects/:id/assignments ────────────────────────

func (h *ProjectHandler) ListAssignments(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	assignments, err := h.svc.ListAssignments(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, id)
	if err != nil {
		respondProjectError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"assignments": assignments, "count": len(assignments)})
}

// ── ListArchive GET /api/projects/archive ─────────────────────────────────────

func (h *ProjectHandler) ListArchive(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	page := 1
	perPage := 25
	if p := c.Query("page"); p != "" {
		if v, err := parseInt(p); err == nil && v > 0 {
			page = v
		}
	}
	if pp := c.Query("per_page"); pp != "" {
		if v, err := parseInt(pp); err == nil && v > 0 {
			perPage = v
		}
	}

	f := services.ArchiveFilter{
		Search:  c.Query("search"),
		Status:  c.Query("status"),
		Page:    page,
		PerPage: perPage,
	}
	if df := c.Query("date_from"); df != "" {
		f.DateFrom = &df
	}
	if dt := c.Query("date_to"); dt != "" {
		f.DateTo = &dt
	}
	if pid := c.Query("poslovoda_id"); pid != "" {
		f.PoslovodaID = pid
	}

	resp, err := h.svc.ListArchive(c.Request.Context(), u.CompanyID, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ── GetArchiveSummary GET /api/projects/:id/archive-summary ──────────────────

func (h *ProjectHandler) GetArchiveSummary(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	summary, err := h.svc.GetArchiveSummary(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, id)
	if err != nil {
		respondProjectError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

func parseInt(s string) (int, error) {
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	return v, err
}

// ── Error helpers ─────────────────────────────────────────────────────────────

func respondProjectHardDeleteError(c *gin.Context, err error) {
	var blocked *repositories.HardDeleteBlockedError
	var ve *services.ValidationError
	switch {
	case errors.As(err, &blocked):
		c.JSON(http.StatusConflict, gin.H{"error": blocked.Reason})
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Projekt nije pronađen"})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Greška pri brisanju projekta"})
	}
}

func respondProjectError(c *gin.Context, err error) {
	var ve *services.ValidationError
	switch {
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	case errors.Is(err, services.ErrProjectNotDeletable):
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrProjectNotActive):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}

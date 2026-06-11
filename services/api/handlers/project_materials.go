package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
	"github.com/gradiliste/api/services"
)

type ProjectMaterialHandler struct {
	svc *services.ProjectMaterialService
}

func NewProjectMaterialHandler(svc *services.ProjectMaterialService) *ProjectMaterialHandler {
	return &ProjectMaterialHandler{svc: svc}
}

// GET /api/projects/:id/materials
func (h *ProjectMaterialHandler) List(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Param("id")

	search := c.Query("search")
	activeOnly := c.Query("active") != "false"

	items, err := h.svc.List(c.Request.Context(), projectID, u.CompanyID, search, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"materials": items, "count": len(items)})
}

// POST /api/projects/:id/materials
func (h *ProjectMaterialHandler) Create(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Param("id")

	var req dto.CreateMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	m, err := h.svc.Create(c.Request.Context(), projectID, u.CompanyID, u.UserID, req)
	if err != nil {
		respondMaterialError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"material": m})
}

// PUT /api/projects/:id/materials/:materialId
func (h *ProjectMaterialHandler) Update(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Param("id")
	materialID := c.Param("materialId")

	var req dto.UpdateMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	m, err := h.svc.Update(c.Request.Context(), materialID, projectID, u.CompanyID, u.UserID, req)
	if err != nil {
		respondMaterialError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"material": m})
}

// PATCH /api/projects/:id/materials/:materialId/deactivate
func (h *ProjectMaterialHandler) Deactivate(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Param("id")
	materialID := c.Param("materialId")

	if err := h.svc.Deactivate(c.Request.Context(), materialID, projectID, u.CompanyID, u.UserID); err != nil {
		respondMaterialError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Material deactivated"})
}

// POST /api/projects/:id/materials/import/analyze
// Accepts a multipart Excel file, detects headers, returns mapping suggestions + sample rows.
func (h *ProjectMaterialHandler) ImportAnalyze(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Param("id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required (multipart field: file)"})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot read uploaded file"})
		return
	}

	resp, err := h.svc.Analyze(c.Request.Context(), projectID, u.CompanyID, u.UserID, header.Filename, data)
	if err != nil {
		respondMaterialError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/projects/:id/materials/import/preview
// Accepts import_job_id + user-selected mapping, validates rows, returns preview.
func (h *ProjectMaterialHandler) ImportWizardPreview(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Param("id")

	var body struct {
		ImportJobID string            `json:"import_job_id" binding:"required"`
		Mapping     map[string]string `json:"mapping" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.WizardPreview(c.Request.Context(), body.ImportJobID, projectID, u.CompanyID, body.Mapping)
	if err != nil {
		respondMaterialError(c, err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// POST /api/projects/:id/materials/import/confirm
// Imports director-reviewed (optionally edited) rows into project_materials.
// The frontend sends the full edited row list; if rows is absent the stored preview is used.
func (h *ProjectMaterialHandler) ImportConfirm(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Param("id")

	var body struct {
		ImportJobID string                 `json:"import_job_id" binding:"required"`
		Rows        []dto.WizardConfirmRow `json:"rows"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.svc.ConfirmImport(c.Request.Context(), body.ImportJobID, projectID, u.CompanyID, u.UserID, body.Rows)
	if err != nil {
		respondMaterialError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func respondMaterialError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repositories.ErrMaterialNotFound), errors.Is(err, repositories.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	case errors.Is(err, repositories.ErrMaterialDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "Materijal s ovim nazivom i jedinicom mjere već postoji na ovom projektu"})
	case errors.Is(err, repositories.ErrImportJobNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Import job not found"})
	case errors.Is(err, services.ErrImportJobConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "Import job is not in the required state (upload and preview first)"})
	case errors.Is(err, services.ErrProjectNotActive):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

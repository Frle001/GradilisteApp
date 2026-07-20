package handlers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/repositories"
	"github.com/gradiliste/api/services"
)

type FolderUploadHandler struct {
	svc        *services.FolderUploadService
	folderRepo *repositories.ProjectFoldersRepository
	docRepo    *repositories.ProjectDocumentsRepository
}

func NewFolderUploadHandler(
	svc *services.FolderUploadService,
	folderRepo *repositories.ProjectFoldersRepository,
	docRepo *repositories.ProjectDocumentsRepository,
) *FolderUploadHandler {
	return &FolderUploadHandler{svc: svc, folderRepo: folderRepo, docRepo: docRepo}
}

// POST /api/projects/:id/documents/folder-upload
// Multipart fields: files[] (files), relative_paths[] (strings, index-matched), root_folder_id (optional)
func (h *FolderUploadHandler) BatchUpload(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	projectID := c.Param("id")

	// ParseMultipartForm with 32 MB in-memory; remainder spills to temp files.
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("neispravan multipart zahtjev: %v", err)})
		return
	}

	form := c.Request.MultipartForm
	fileHeaders := form.File["files"]
	relativePaths := form.Value["relative_paths"]

	if len(fileHeaders) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nema datoteka (polje 'files')"})
		return
	}
	if len(relativePaths) != len(fileHeaders) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("broj relative_paths (%d) ne podudara se s brojem datoteka (%d)",
				len(relativePaths), len(fileHeaders)),
		})
		return
	}

	// Optional root folder.
	rootFolderIDVal := c.PostForm("root_folder_id")
	var rootFolderID *string
	if rootFolderIDVal != "" {
		rootFolderID = &rootFolderIDVal
	}

	// Build input slice — open each file.
	// Collect closers explicitly; defer inside a loop does not close until function return.
	var openFiles []io.Closer
	defer func() {
		for _, f := range openFiles {
			f.Close()
		}
	}()

	inputs := make([]services.BatchUploadInput, 0, len(fileHeaders))
	for i, fh := range fileHeaders {
		f, err := fh.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("ne mogu otvoriti datoteku %d: %v", i, err)})
			return
		}
		openFiles = append(openFiles, f)
		inputs = append(inputs, services.BatchUploadInput{
			File:         f,
			Header:       fh,
			RelativePath: relativePaths[i],
		})
	}

	resp, err := h.svc.Process(c.Request.Context(), u.CompanyID, projectID, u.UserID, rootFolderID, inputs)
	if err != nil {
		if errors.Is(err, services.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "projekt ili mapa nije pronađena"})
			return
		}
		if errors.Is(err, services.ErrForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "mapa ne pripada ovom projektu"})
			return
		}
		if ve := services.AsValidationError(err); ve != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ve.Error()})
			return
		}
		log.Printf("[folder_upload] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri batch uploadu"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GET /api/projects/:id/folders?parent_id=xxx
func (h *FolderUploadHandler) ListFolders(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	projectID := c.Param("id")

	// Verify project belongs to company.
	exists, err := h.docRepo.ProjectBelongsToCompany(c.Request.Context(), u.CompanyID, projectID)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "projekt nije pronađen"})
		return
	}

	parentIDStr := c.Query("parent_id")
	var parentID *string
	if parentIDStr != "" {
		parentID = &parentIDStr
		// Verify parent folder belongs to project.
		ok, err := h.folderRepo.BelongsToProject(c.Request.Context(), u.CompanyID, projectID, parentIDStr)
		if err != nil || !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "mapa nije pronađena"})
			return
		}
	}

	folders, err := h.folderRepo.ListByParent(c.Request.Context(), u.CompanyID, projectID, parentID)
	if err != nil {
		log.Printf("[folders] ListFolders error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri dohvatu mapa"})
		return
	}

	items := make([]dto.ProjectFolderItem, 0, len(folders))
	for _, f := range folders {
		items = append(items, dto.ProjectFolderItem{
			ID:        f.ID,
			Name:      f.Name,
			ParentID:  f.ParentID,
			CreatedAt: f.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, gin.H{"folders": items})
}

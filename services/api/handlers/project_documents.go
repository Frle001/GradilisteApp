package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/services"
)

type ProjectDocumentsHandler struct {
	svc *services.ProjectDocumentsService
}

func NewProjectDocumentsHandler(svc *services.ProjectDocumentsService) *ProjectDocumentsHandler {
	return &ProjectDocumentsHandler{svc: svc}
}

// GET /api/projects/:id/documents
func (h *ProjectDocumentsHandler) List(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	projectID := c.Param("id")

	docs, err := h.svc.List(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, projectID)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "projekt nije pronađen"})
			return
		}
		if errors.Is(err, services.ErrDocumentForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "pristup odbijen"})
			return
		}
		log.Printf("[project_documents] List error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri dohvatu dokumenata"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"documents": docs})
}

// POST /api/projects/:id/documents  (multipart/form-data, field "file")
func (h *ProjectDocumentsHandler) Upload(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	projectID := c.Param("id")

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datoteka je obavezna (polje 'file')"})
		return
	}
	defer file.Close()

	doc, err := h.svc.Upload(c.Request.Context(), u.CompanyID, u.UserID, projectID, file, header)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "projekt nije pronađen"})
			return
		}
		log.Printf("[project_documents] Upload error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"document": doc})
}

// GET /api/projects/:id/documents/:docId/download
func (h *ProjectDocumentsHandler) Download(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	projectID := c.Param("id")
	docID := c.Param("docId")

	dl, err := h.svc.Download(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, projectID, docID)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dokument nije pronađen"})
			return
		}
		if errors.Is(err, services.ErrDocumentForbidden) {
			c.JSON(http.StatusForbidden, gin.H{"error": "pristup odbijen"})
			return
		}
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(http.StatusNotFound, gin.H{"error": "datoteka nije pronađena na poslužitelju"})
			return
		}
		log.Printf("[project_documents] Download error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri preuzimanju dokumenta"})
		return
	}
	defer dl.Reader.Close()

	disposition := fmt.Sprintf(`attachment; filename="%s"`, dl.OrigName)
	c.DataFromReader(http.StatusOK, dl.FileSize, dl.ContentType, dl.Reader, map[string]string{
		"Content-Disposition": disposition,
	})
}

// DELETE /api/projects/:id/documents/:docId
func (h *ProjectDocumentsHandler) Delete(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	projectID := c.Param("id")
	docID := c.Param("docId")

	err := h.svc.Delete(c.Request.Context(), u.CompanyID, projectID, docID)
	if err != nil {
		if errors.Is(err, services.ErrDocumentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "dokument nije pronađen"})
			return
		}
		log.Printf("[project_documents] Delete error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri brisanju dokumenta"})
		return
	}
	c.Status(http.StatusNoContent)
}

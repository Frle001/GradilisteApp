package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/services"
)

type DailyReportAttachmentsHandler struct {
	svc *services.DailyReportAttachmentsService
}

func NewDailyReportAttachmentsHandler(svc *services.DailyReportAttachmentsService) *DailyReportAttachmentsHandler {
	return &DailyReportAttachmentsHandler{svc: svc}
}

// GET /api/daily-reports/:id/attachments
func (h *DailyReportAttachmentsHandler) List(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	reportID := c.Param("id")

	attachments, err := h.svc.List(c.Request.Context(), u.CompanyID, reportID)
	if err != nil {
		if errors.Is(err, services.ErrAttachmentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "izvještaj nije pronađen"})
			return
		}
		log.Printf("[report_attachments] List error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri dohvatu privitaka"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"attachments": attachments})
}

// POST /api/daily-reports/:id/attachments  (multipart/form-data, field "photo")
func (h *DailyReportAttachmentsHandler) Upload(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	reportID := c.Param("id")

	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slika je obavezna (polje 'photo')"})
		return
	}
	defer file.Close()

	att, err := h.svc.Upload(
		c.Request.Context(),
		u.CompanyID, reportID, u.UserID, u.Role, u.EmployeeID,
		file, header,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAttachmentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "izvještaj nije pronađen"})
		case errors.Is(err, services.ErrAttachmentForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "pristup odbijen"})
		case errors.Is(err, services.ErrTooManyAttachments):
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		case errors.Is(err, services.ErrDailyReportNotEditable):
			c.JSON(http.StatusConflict, gin.H{"error": "odobren izvještaj ne može se mijenjati"})
		default:
			log.Printf("[report_attachments] Upload error: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"attachment": att})
}

// GET /api/daily-reports/:id/attachments/:attachId/download
func (h *DailyReportAttachmentsHandler) Download(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	reportID := c.Param("id")
	attachID := c.Param("attachId")

	dl, err := h.svc.Download(c.Request.Context(), u.CompanyID, reportID, attachID)
	if err != nil {
		if errors.Is(err, services.ErrAttachmentNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "privitak nije pronađen"})
			return
		}
		log.Printf("[report_attachments] Download error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri preuzimanju privitka"})
		return
	}
	defer dl.Reader.Close()

	disposition := fmt.Sprintf(`inline; filename="%s"`, dl.OrigName)
	c.DataFromReader(http.StatusOK, dl.FileSize, dl.ContentType, dl.Reader, map[string]string{
		"Content-Disposition": disposition,
	})
}

// DELETE /api/daily-reports/:id/attachments/:attachId
func (h *DailyReportAttachmentsHandler) Delete(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	if u == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	reportID := c.Param("id")
	attachID := c.Param("attachId")

	err := h.svc.Delete(c.Request.Context(), u.CompanyID, reportID, attachID, u.Role, u.EmployeeID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAttachmentNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "privitak nije pronađen"})
		case errors.Is(err, services.ErrAttachmentForbidden):
			c.JSON(http.StatusForbidden, gin.H{"error": "pristup odbijen"})
		case errors.Is(err, services.ErrDailyReportNotEditable):
			c.JSON(http.StatusConflict, gin.H{"error": "odobren izvještaj ne može se mijenjati"})
		default:
			log.Printf("[report_attachments] Delete error: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "greška pri brisanju privitka"})
		}
		return
	}
	c.Status(http.StatusNoContent)
}

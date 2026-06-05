package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/dto"
	"github.com/gradiliste/api/services"
)

type ReportsHandler struct {
	svc *services.ReportsService
}

func NewReportsHandler(svc *services.ReportsService) *ReportsHandler {
	return &ReportsHandler{svc: svc}
}

// parseReportFilter reads all common filter query params into a ReportFilter.
func parseReportFilter(c *gin.Context) dto.ReportFilter {
	f := dto.ReportFilter{}
	if v := c.Query("project_id"); v != "" {
		f.ProjectID = &v
	}
	if v := c.Query("poslovoda_id"); v != "" {
		f.PoslovodaID = &v
	}
	if v := c.Query("worker_id"); v != "" {
		f.WorkerID = &v
	}
	if v := c.Query("material_id"); v != "" {
		f.MaterialID = &v
	}
	if v := c.Query("activity_type"); v != "" {
		f.ActivityType = &v
	}
	if v := c.Query("is_vtk"); v != "" {
		b := v == "true" || v == "1"
		f.IsVTK = &b
	}
	if v := c.Query("date_from"); v != "" {
		f.DateFrom = &v
	}
	if v := c.Query("date_to"); v != "" {
		f.DateTo = &v
	}
	if v := c.Query("status"); v != "" {
		f.Status = &v
	}
	if v := c.Query("search"); v != "" {
		f.Search = &v
	}
	return f
}

func parsePagination(c *gin.Context) (page, perPage int) {
	page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ = strconv.Atoi(c.DefaultQuery("per_page", "25"))
	return
}

// GET /api/reports/gradevinski-dnevnik
func (h *ReportsHandler) ListDnevnik(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	f := parseReportFilter(c)
	page, perPage := parsePagination(c)

	log.Printf("[reports] ListDnevnik company=%s role=%s emp=%s page=%d perPage=%d",
		u.CompanyID, u.Role, u.EmployeeID, page, perPage)

	resp, err := h.svc.ListDnevnik(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, f, page, perPage)
	if err != nil {
		log.Printf("[reports] ListDnevnik error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/reports/gradevinski-dnevnik/export
func (h *ReportsHandler) ExportDnevnik(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	f := parseReportFilter(c)

	buf, err := h.svc.ExportDnevnik(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, f)
	if err != nil {
		if errors.Is(err, services.ErrExportLimitExceeded) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[reports] ExportDnevnik error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := services.ExcelFilename("gradjevinski-dnevnik")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// GET /api/reports/gradevinska-knjiga
func (h *ReportsHandler) ListKnjiga(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	f := parseReportFilter(c)
	page, perPage := parsePagination(c)

	resp, err := h.svc.ListKnjiga(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, f, page, perPage)
	if err != nil {
		log.Printf("[reports] ListKnjiga error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// GET /api/reports/gradevinska-knjiga/export
func (h *ReportsHandler) ExportKnjiga(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	f := parseReportFilter(c)

	buf, err := h.svc.ExportKnjiga(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, f)
	if err != nil {
		if errors.Is(err, services.ErrExportLimitExceeded) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[reports] ExportKnjiga error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := services.ExcelFilename("gradevinska-knjiga")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// GET /api/reports/filter-options
func (h *ReportsHandler) FilterOptions(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	opts, err := h.svc.GetFilterOptions(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID)
	if err != nil {
		log.Printf("[reports] FilterOptions error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, opts)
}

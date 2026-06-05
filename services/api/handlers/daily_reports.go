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

type DailyReportHandler struct {
	svc *services.DailyReportService
}

func NewDailyReportHandler(svc *services.DailyReportService) *DailyReportHandler {
	return &DailyReportHandler{svc: svc}
}

// GET /api/daily-reports
func (h *DailyReportHandler) List(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	f := dto.DailyReportFilter{}
	if v := c.Query("project_id"); v != "" {
		f.ProjectID = &v
	}
	if v := c.Query("poslovoda_id"); v != "" {
		f.PoslovodaID = &v
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

	items, err := h.svc.List(c.Request.Context(), u.CompanyID, u.Role, u.EmployeeID, f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"daily_reports": items, "count": len(items)})
}

// GET /api/daily-reports/form-data
func (h *DailyReportHandler) GetFormData(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	projectID := c.Query("project_id")

	data, err := h.svc.GetFormData(c.Request.Context(), u.CompanyID, u.EmployeeID, u.Role, projectID)
	if err != nil {
		respondDailyReportError(c, err)
		return
	}
	c.JSON(http.StatusOK, data)
}

// GET /api/daily-reports/:id
func (h *DailyReportHandler) GetByID(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	report, err := h.svc.GetByID(c.Request.Context(), id, u.CompanyID, u.Role, u.EmployeeID)
	if err != nil {
		respondDailyReportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"daily_report": report})
}

// POST /api/daily-reports
func (h *DailyReportHandler) Create(c *gin.Context) {
	u := appctx.GetAuthUser(c)

	var req dto.CreateDailyReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.svc.Create(c.Request.Context(), u.CompanyID, u.UserID, u.EmployeeID, u.Role, req)
	if err != nil {
		respondDailyReportError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// PUT /api/daily-reports/:id
func (h *DailyReportHandler) Update(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	var req dto.CreateDailyReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.svc.Update(c.Request.Context(), id, u.CompanyID, u.UserID, u.EmployeeID, u.Role, req)
	if err != nil {
		respondDailyReportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"daily_report": report})
}

// PATCH /api/daily-reports/:id/approve
func (h *DailyReportHandler) Approve(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	if err := h.svc.Approve(c.Request.Context(), id, u.CompanyID, u.UserID, u.EmployeeID); err != nil {
		respondDailyReportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Izvještaj odobren"})
}

// PATCH /api/daily-reports/:id/reject
func (h *DailyReportHandler) Reject(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	id := c.Param("id")

	var req dto.RejectDailyReportRequest
	_ = c.ShouldBindJSON(&req) // reason is optional

	if err := h.svc.Reject(c.Request.Context(), id, u.CompanyID, u.UserID, u.EmployeeID, req.Reason); err != nil {
		respondDailyReportError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Izvještaj odbijen"})
}

// DELETE /api/daily-reports/:id
func (h *DailyReportHandler) Delete(c *gin.Context) {
	c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "Daily reports are not permanently deleted."})
}

// ── Error mapping ─────────────────────────────────────────────────────────────

func respondDailyReportError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repositories.ErrDailyReportNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Dnevni izvještaj nije pronađen"})
	case errors.Is(err, repositories.ErrDailyReportDuplicate):
		c.JSON(http.StatusConflict, gin.H{"error": "Izvještaj za ovaj projekt, poslovođu i datum već postoji"})
	case errors.Is(err, services.ErrDailyReportForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Nemate pristup ovom izvještaju"})
	case errors.Is(err, services.ErrDailyReportNotEditable):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Odobreni izvještaj se ne može mijenjati"})
	case errors.Is(err, services.ErrDailyReportInvalidStatus):
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Izvještaj nije u stanju koje dozvoljava ovu radnju"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}
}

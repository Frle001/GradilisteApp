package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/appctx"
	"github.com/gradiliste/api/services"
)

type DocumentationHandler struct {
	svc *services.DocumentationService
}

func NewDocumentationHandler(svc *services.DocumentationService) *DocumentationHandler {
	return &DocumentationHandler{svc: svc}
}

// ── Employees ─────────────────────────────────────────────────────────────────

// GET /api/documentation/employees
func (h *DocumentationHandler) ListEmployees(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	emps, err := h.svc.ListEmployees(c.Request.Context(), u.CompanyID, u.Role)
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"employees": emps})
}

// GET /api/documentation/employees/:employeeId/summary
func (h *DocumentationHandler) GetEmployeeSummary(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	summary, err := h.svc.GetEmployeeSummary(c.Request.Context(), u.CompanyID, u.Role, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": summary})
}

// ── Notifications ─────────────────────────────────────────────────────────────

// GET /api/documentation/notifications
func (h *DocumentationHandler) GetNotifications(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	notifs, err := h.svc.GetNotifications(c.Request.Context(), u.CompanyID, u.Role, time.Now())
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifs, "total": len(notifs)})
}

// ── File upload ───────────────────────────────────────────────────────────────

// POST /api/documentation/employees/:employeeId/files
func (h *DocumentationHandler) UploadFile(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datoteka je obavezna"})
		return
	}
	defer file.Close()

	fileID, err := h.svc.UploadDocFile(c.Request.Context(), u.CompanyID, u.Role, u.UserID, c.Param("employeeId"), file, header)
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"file_id": fileID})
}

// ── Medical exams ─────────────────────────────────────────────────────────────

// POST /api/documentation/employees/:employeeId/medical-exams
func (h *DocumentationHandler) CreateMedicalExam(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		CompletedDate string `json:"completed_date"`
		ExpiresAt     string `json:"expires_at"`
		DocumentID    string `json:"document_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	exam, err := h.svc.CreateMedicalExam(c.Request.Context(), u.CompanyID, u.Role, u.UserID, services.CreateMedicalExamRequest{
		EmployeeID:    c.Param("employeeId"),
		CompletedDate: body.CompletedDate,
		ExpiresAt:     body.ExpiresAt,
		DocumentID:    body.DocumentID,
	})
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"exam": exam})
}

// GET /api/documentation/employees/:employeeId/medical-exams
func (h *DocumentationHandler) ListMedicalExams(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	exams, err := h.svc.ListMedicalExams(c.Request.Context(), u.CompanyID, u.Role, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"exams": exams})
}

// ── Safety records ────────────────────────────────────────────────────────────

// PUT /api/documentation/employees/:employeeId/safety
func (h *DocumentationHandler) UploadSafetyRecord(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		DocumentID string `json:"document_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rec, err := h.svc.UploadSafetyRecord(c.Request.Context(), u.CompanyID, u.Role, u.UserID, c.Param("employeeId"), body.DocumentID)
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

// POST /api/documentation/employees/:employeeId/safety/verify
func (h *DocumentationHandler) VerifySafetyRecord(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	rec, err := h.svc.VerifySafetyRecord(c.Request.Context(), u.CompanyID, u.Role, u.UserID, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

// GET /api/documentation/employees/:employeeId/safety
func (h *DocumentationHandler) GetSafetyRecord(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	rec, err := h.svc.GetSafetyRecord(c.Request.Context(), u.CompanyID, u.Role, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

// ── Contracts ─────────────────────────────────────────────────────────────────

// POST /api/documentation/employees/:employeeId/contracts
func (h *DocumentationHandler) CreateContract(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		ContractType string  `json:"contract_type"`
		ExpiresAt    *string `json:"expires_at"`
		DocumentID   string  `json:"document_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	contract, err := h.svc.CreateContract(c.Request.Context(), u.CompanyID, u.Role, u.UserID, services.CreateContractRequest{
		EmployeeID:   c.Param("employeeId"),
		ContractType: body.ContractType,
		ExpiresAt:    body.ExpiresAt,
		DocumentID:   body.DocumentID,
	})
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"contract": contract})
}

// GET /api/documentation/employees/:employeeId/contracts
func (h *DocumentationHandler) ListContracts(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	contracts, err := h.svc.ListContracts(c.Request.Context(), u.CompanyID, u.Role, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"contracts": contracts})
}

// POST /api/documentation/contracts/:contractId/terminate
func (h *DocumentationHandler) TerminateContract(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		TerminatedAt     string `json:"terminated_at"`
		TerminationDocID string `json:"termination_document_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.svc.TerminateContract(c.Request.Context(), u.CompanyID, u.Role, u.UserID, services.TerminateContractRequest{
		ContractID:       c.Param("contractId"),
		TerminatedAt:     body.TerminatedAt,
		TerminationDocID: body.TerminationDocID,
	})
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── Monthly obligations ───────────────────────────────────────────────────────

// POST /api/documentation/employees/:employeeId/payroll
func (h *DocumentationHandler) CompletePayroll(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		Year       int     `json:"year"`
		Month      int     `json:"month"`
		DocumentID *string `json:"document_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	obl, err := h.svc.CompletePayroll(c.Request.Context(), u.CompanyID, u.Role, u.UserID, services.CompletePayrollRequest{
		EmployeeID: c.Param("employeeId"),
		Year:       body.Year,
		Month:      body.Month,
		DocumentID: body.DocumentID,
	})
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"obligation": obl})
}

// POST /api/documentation/employees/:employeeId/timesheet
func (h *DocumentationHandler) CompleteTimesheet(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		Year  int `json:"year"`
		Month int `json:"month"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	obl, err := h.svc.CompleteTimesheet(c.Request.Context(), u.CompanyID, u.Role, u.UserID, services.CompleteTimesheetRequest{
		EmployeeID: c.Param("employeeId"),
		Year:       body.Year,
		Month:      body.Month,
	})
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"obligation": obl})
}

// GET /api/documentation/employees/:employeeId/obligations
func (h *DocumentationHandler) ListMonthlyObligations(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	obls, err := h.svc.ListMonthlyObligations(c.Request.Context(), u.CompanyID, u.Role, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"obligations": obls})
}

// ── Annual leave ──────────────────────────────────────────────────────────────

// POST /api/documentation/employees/:employeeId/annual-leave
func (h *DocumentationHandler) CreateAnnualLeave(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		DateFrom   *string `json:"date_from"`
		DateTo     *string `json:"date_to"`
		DocumentID *string `json:"document_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	leave, err := h.svc.CreateAnnualLeave(c.Request.Context(), u.CompanyID, u.Role, u.UserID, services.CreateAnnualLeaveRequest{
		EmployeeID: c.Param("employeeId"),
		DateFrom:   body.DateFrom,
		DateTo:     body.DateTo,
		DocumentID: body.DocumentID,
	})
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"leave": leave})
}

// GET /api/documentation/employees/:employeeId/annual-leave
func (h *DocumentationHandler) ListAnnualLeave(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	leaves, err := h.svc.ListAnnualLeave(c.Request.Context(), u.CompanyID, u.Role, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"leaves": leaves})
}

// ── Work permits ──────────────────────────────────────────────────────────────

// POST /api/documentation/employees/:employeeId/work-permits
func (h *DocumentationHandler) CreateWorkPermit(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		EnteredAt  string `json:"entered_at"`
		ExpiresAt  string `json:"expires_at"`
		DocumentID string `json:"document_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	permit, err := h.svc.CreateWorkPermit(c.Request.Context(), u.CompanyID, u.Role, u.UserID, services.CreateWorkPermitRequest{
		EmployeeID: c.Param("employeeId"),
		EnteredAt:  body.EnteredAt,
		ExpiresAt:  body.ExpiresAt,
		DocumentID: body.DocumentID,
	})
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"permit": permit})
}

// GET /api/documentation/employees/:employeeId/work-permits
func (h *DocumentationHandler) ListWorkPermits(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	permits, err := h.svc.ListWorkPermits(c.Request.Context(), u.CompanyID, u.Role, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"permits": permits})
}

// ── Pension ───────────────────────────────────────────────────────────────────

// PUT /api/documentation/employees/:employeeId/pension
func (h *DocumentationHandler) UploadPensionRecord(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		DocumentID string `json:"document_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rec, err := h.svc.UploadPensionRecord(c.Request.Context(), u.CompanyID, u.Role, u.UserID, c.Param("employeeId"), body.DocumentID)
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

// POST /api/documentation/employees/:employeeId/pension/verify
func (h *DocumentationHandler) VerifyPensionRecord(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	rec, err := h.svc.VerifyPensionRecord(c.Request.Context(), u.CompanyID, u.Role, u.UserID, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

// ── Health ────────────────────────────────────────────────────────────────────

// PUT /api/documentation/employees/:employeeId/health
func (h *DocumentationHandler) UploadHealthRecord(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	var body struct {
		DocumentID string `json:"document_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rec, err := h.svc.UploadHealthRecord(c.Request.Context(), u.CompanyID, u.Role, u.UserID, c.Param("employeeId"), body.DocumentID)
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

// POST /api/documentation/employees/:employeeId/health/verify
func (h *DocumentationHandler) VerifyHealthRecord(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	rec, err := h.svc.VerifyHealthRecord(c.Request.Context(), u.CompanyID, u.Role, u.UserID, c.Param("employeeId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"record": rec})
}

// GET /api/documentation/files/:fileId/download
func (h *DocumentationHandler) DownloadDocFile(c *gin.Context) {
	u := appctx.GetAuthUser(c)
	reader, ct, filename, err := h.svc.DownloadDocFile(c.Request.Context(), u.CompanyID, u.Role, c.Param("fileId"))
	if err != nil {
		respondDocError(c, err)
		return
	}
	defer reader.Close()
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.DataFromReader(http.StatusOK, -1, ct, reader, nil)
}

// ── Error mapping ─────────────────────────────────────────────────────────────

func respondDocError(c *gin.Context, err error) {
	var ve *services.ValidationError
	switch {
	case errors.Is(err, services.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	case errors.Is(err, services.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	default:
		log.Printf("[documentation] error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
	}
}

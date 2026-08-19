package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

func RegisterDocumentationRoutes(
	api *gin.RouterGroup,
	h *handlers.DocumentationHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	adminOnly := requireRoles("administracija")

	rg := api.Group("/documentation", authRequired, adminOnly)
	{
		rg.GET("/employees", h.ListEmployees)
		rg.GET("/notifications", h.GetNotifications)

		emp := rg.Group("/employees/:employeeId")
		{
			emp.GET("/summary", h.GetEmployeeSummary)
			emp.POST("/files", h.UploadFile)

			// Liječnički pregled
			emp.POST("/medical-exams", h.CreateMedicalExam)
			emp.GET("/medical-exams", h.ListMedicalExams)

			// Zaštita na radu
			emp.PUT("/safety", h.UploadSafetyRecord)
			emp.GET("/safety", h.GetSafetyRecord)
			emp.POST("/safety/verify", h.VerifySafetyRecord)

			// Ugovor o radu
			emp.POST("/contracts", h.CreateContract)
			emp.GET("/contracts", h.ListContracts)

			// Platne liste
			emp.POST("/payroll", h.CompletePayroll)

			// Evidencija radnika
			emp.POST("/timesheet", h.CompleteTimesheet)

			// Monthly obligations (combined list)
			emp.GET("/obligations", h.ListMonthlyObligations)

			// Godišnji odmor
			emp.POST("/annual-leave", h.CreateAnnualLeave)
			emp.GET("/annual-leave", h.ListAnnualLeave)

			// Radne dozvole
			emp.POST("/work-permits", h.CreateWorkPermit)
			emp.GET("/work-permits", h.ListWorkPermits)

			// Mirovinsko
			emp.PUT("/pension", h.UploadPensionRecord)
			emp.POST("/pension/verify", h.VerifyPensionRecord)

			// Zdravstveno
			emp.PUT("/health", h.UploadHealthRecord)
			emp.POST("/health/verify", h.VerifyHealthRecord)
		}

		// Contract termination (contract-level, not employee-level)
		rg.POST("/contracts/:contractId/terminate", h.TerminateContract)

		// File download by file ID (company-scoped)
		rg.GET("/files/:fileId/download", h.DownloadDocFile)
	}
}

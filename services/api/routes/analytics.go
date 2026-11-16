package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/handlers"
)

func RegisterAnalyticsRoutes(api *gin.RouterGroup, h *handlers.AnalyticsHandler, coldOps *handlers.ColdOpsHandler, revenue *handlers.RevenueHandler, authRequired gin.HandlerFunc, requireRoles func(...string) gin.HandlerFunc) {
	dirOrInz := requireRoles("direktor", "inzenjer")
	analytics := api.Group("/analytics", authRequired, dirOrInz)
	{
		place := analytics.Group("/place")
		{
			place.GET("/summary", h.GetMonthlyLaborSummary)
			place.GET("/employees/:employeeId", h.GetEmployeeLaborCost)
			place.GET("/employees/:employeeId/compensation", h.ListCompensationPlans)
			place.POST("/employees/:employeeId/compensation", h.CreateCompensationPlan)
			place.GET("/annual", h.GetAnnualPayrollSummary)
			place.GET("/project-allocation", h.GetMonthlyProjectAllocation)
			place.PUT("/employees/:employeeId/contributions", h.UpsertContribution)
			place.GET("/employees/:employeeId/history", h.GetEmployeePayrollHistory)
			place.PATCH("/compensation/:planId", h.UpdateCompensationPlan)
		}

		projects := analytics.Group("/projects")
		{
			projects.GET("/:projectId", h.GetProjectDetail)
		}

		co := analytics.Group("/hladni-pogon")
		{
			co.GET("/summary", coldOps.GetSummary)
			co.GET("/annual", coldOps.GetAnnualSummary)

			co.GET("/categories", coldOps.ListCategories)
			co.POST("/categories", coldOps.CreateCategory)
			co.PUT("/categories/:id", coldOps.UpdateCategory)
			co.DELETE("/categories/:id", coldOps.DeleteCategory)

			co.GET("/expenses", coldOps.ListExpenses)
			co.POST("/expenses", coldOps.CreateExpense)
			co.GET("/expenses/:id", coldOps.GetExpense)
			co.PUT("/expenses/:id", coldOps.UpdateExpense)
			co.DELETE("/expenses/:id", coldOps.DeleteExpense)

			co.GET("/recurring", coldOps.ListRecurringDefs)
			co.POST("/recurring", coldOps.CreateRecurringDef)
			co.PUT("/recurring/:id", coldOps.UpdateRecurringDef)
			co.GET("/recurring/:defId/occurrences", coldOps.ListOccurrences)
			co.POST("/occurrences/:id/confirm", coldOps.ConfirmOccurrence)
			co.POST("/occurrences/:id/skip", coldOps.SkipOccurrence)

			co.GET("/budgets", coldOps.ListBudgets)
			co.PUT("/budgets", coldOps.UpsertBudget)
			co.DELETE("/budgets/:id", coldOps.DeleteBudget)
		}

		prihodi := analytics.Group("/prihodi")
		{
			prihodi.GET("/summary", revenue.GetRevenueSummary)
			prihodi.GET("/projects/:projectId", revenue.GetProjectRevenueDetail)

			prihodi.GET("/invoices", revenue.ListInvoices)
			prihodi.POST("/invoices", revenue.CreateInvoice)
			prihodi.GET("/invoices/:id", revenue.GetInvoice)
			prihodi.POST("/invoices/:id/cancel", revenue.CancelInvoice)
			prihodi.POST("/invoices/:id/payments", revenue.AddPayment)
			prihodi.GET("/invoices/:id/payments", revenue.ListPayments)

			prihodi.GET("/contracts", revenue.ListContracts)
			prihodi.POST("/contracts", revenue.CreateContract)
			prihodi.GET("/contracts/:id", revenue.GetContract)
			prihodi.PUT("/contracts/:id", revenue.UpdateContract)
			prihodi.DELETE("/contracts/:id", revenue.DeleteContract)
		}
	}
}

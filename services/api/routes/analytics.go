package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/gradiliste/api/handlers"
)

func RegisterAnalyticsRoutes(api *gin.RouterGroup, h *handlers.AnalyticsHandler, authRequired gin.HandlerFunc, requireRoles func(...string) gin.HandlerFunc) {
	dirOrInz := requireRoles("direktor", "inzenjer")
	analytics := api.Group("/analytics", authRequired, dirOrInz)
	{
		place := analytics.Group("/place")
		{
			place.GET("/summary", h.GetMonthlyLaborSummary)
			place.GET("/employees/:employeeId", h.GetEmployeeLaborCost)
			place.GET("/employees/:employeeId/compensation", h.ListCompensationPlans)
			place.POST("/employees/:employeeId/compensation", h.CreateCompensationPlan)
			place.PATCH("/compensation/:planId", h.UpdateCompensationPlan)
		}
	}
}

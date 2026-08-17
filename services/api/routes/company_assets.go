package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

func RegisterCompanyAssetsRoutes(
	api *gin.RouterGroup,
	h *handlers.CompanyAssetsHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	adminOnly := requireRoles("administracija")

	// Static sub-paths must be registered before /:id to prevent Gin treating
	// "employees" and "notifications" as an :id parameter.
	rg := api.Group("/company-assets", authRequired, adminOnly)
	{
		rg.GET("", h.List)
		rg.POST("", h.Create)
		rg.GET("/employees", h.FormEmployees)
		rg.GET("/notifications", h.Notifications)
		rg.GET("/:id", h.GetByID)
		rg.PATCH("/:id", h.Update)
		rg.DELETE("/:id", h.Deactivate)
		rg.POST("/:id/leasing-payments", h.MarkLeasingPayment)
		rg.GET("/:id/leasing-payments", h.GetLeasingHistory)
	}
}

package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

// RegisterEmployeeRoutes mounts all employee and asset endpoints on the given router group.
// authRequired and requireRoles are passed from package main to avoid import cycles.
func RegisterEmployeeRoutes(
	api *gin.RouterGroup,
	h *handlers.EmployeeHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(roles ...string) gin.HandlerFunc,
) {
	manageRoles := requireRoles("direktor", "inzenjer", "administracija")

	employees := api.Group("/employees", authRequired)
	{
		employees.GET("", h.List)
		employees.GET("/:id", h.GetByID)
		employees.POST("", manageRoles, h.Create)
		employees.PUT("/:id", manageRoles, h.Update)
		employees.PATCH("/:id/deactivate", manageRoles, h.Deactivate)
		employees.PATCH("/:id/activate", manageRoles, h.Activate)

		employees.GET("/:id/assets", h.ListAssets)
		employees.POST("/:id/assets", manageRoles, h.CreateAsset)
		employees.PUT("/:id/assets/:asset_id", manageRoles, h.UpdateAsset)
		employees.PATCH("/:id/assets/:asset_id/deactivate", manageRoles, h.DeactivateAsset)
	}
}

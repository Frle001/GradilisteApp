package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

func RegisterManagementHoursRoutes(
	api *gin.RouterGroup,
	h *handlers.ManagementHoursHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	mgmt := requireRoles("direktor", "inzenjer", "administracija")

	mh := api.Group("/management-hours", authRequired, mgmt)
	{
		mh.POST("", h.Submit)
		mh.GET("", h.List)
		mh.GET("/:id", h.GetByID)
		mh.PUT("/:id", h.Update)
		mh.DELETE("/:id", h.Delete)
	}
}

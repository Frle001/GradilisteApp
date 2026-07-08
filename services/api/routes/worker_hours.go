package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

func RegisterWorkerHoursRoutes(
	api *gin.RouterGroup,
	h *handlers.WorkerHoursHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	radnikOnly := requireRoles("radnik")

	wh := api.Group("/worker-hours", authRequired, radnikOnly)
	{
		wh.GET("/projects", h.ListProjects)
		wh.GET("/my", h.ListMy)
		wh.POST("", h.Submit)
	}
}

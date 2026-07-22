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
	workerOrPoslovoda := requireRoles("radnik", "poslovoda")

	wh := api.Group("/worker-hours", authRequired, workerOrPoslovoda)
	{
		wh.GET("/projects", h.ListProjects)
		wh.GET("/my", h.ListMy)
		wh.GET("/history", h.History)
		wh.POST("", h.Submit)
	}
}

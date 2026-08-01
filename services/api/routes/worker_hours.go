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
	managerOnly := requireRoles("direktor", "inzenjer")

	// Radnik / poslovoda — self-submission routes
	wh := api.Group("/worker-hours", authRequired, workerOrPoslovoda)
	{
		wh.GET("/projects", h.ListProjects)
		wh.GET("/my", h.ListMy)
		wh.GET("/history", h.History)
		wh.POST("", h.Submit)
	}

	// Direktor / inženjer — manager overview and corrections
	mgr := api.Group("/worker-hours/manager", authRequired, managerOnly)
	{
		mgr.GET("", h.ManagerList)
		mgr.GET("/:id", h.ManagerDetail)
		mgr.POST("/:id/correct", h.ManagerCorrect)
		mgr.POST("/:id/comments", h.ManagerAddComment)
	}
}

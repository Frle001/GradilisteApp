package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/gradiliste/api/handlers"
)

func RegisterScheduleRoutes(
	api *gin.RouterGroup,
	h *handlers.ScheduleHandler,
	authRequired gin.HandlerFunc,
	requireRoles func(...string) gin.HandlerFunc,
) {
	viewRoles := requireRoles("direktor", "inzenjer", "administracija", "poslovoda", "radnik")
	manageRoles := requireRoles("direktor", "inzenjer")

	schedule := api.Group("/schedule", authRequired)
	{
		// Read — all authenticated roles
		schedule.GET("/shifts", viewRoles, h.ListShifts)
		schedule.GET("/shifts/:shiftId", viewRoles, h.GetShift)
		schedule.GET("/employees-for-date", viewRoles, h.EmployeesForDate)

		// Management — direktor / inženjer only
		schedule.POST("/shifts", manageRoles, h.CreateShift)
		schedule.PATCH("/shifts/:shiftId", manageRoles, h.UpdateShift)
		schedule.POST("/shifts/:shiftId/cancel", manageRoles, h.CancelShift)
		schedule.PUT("/shifts/:shiftId/assignments", manageRoles, h.SyncAssignments)
		schedule.POST("/shifts/:shiftId/duplicate", manageRoles, h.DuplicateShift)
		schedule.POST("/copy-day", manageRoles, h.CopyDay)
		schedule.POST("/copy-week", manageRoles, h.CopyWeek)
	}
}

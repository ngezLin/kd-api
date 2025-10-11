package routes

import (
	"kd-api/controllers"
	"kd-api/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {

	r.POST("/login", controllers.Login)

	// Public items buat landing page
	r.GET("/public/items", controllers.GetItems)
	r.GET("/public/items/:id", controllers.GetItemByID)

	// Items 
	items := r.Group("/items")
	items.Use(middlewares.AuthMiddleware())
	{
		items.GET("/", controllers.GetItems)          
		items.GET("/:id", controllers.GetItemByID)     
		items.POST("/", middlewares.RoleMiddleware("admin", "cashier"), controllers.CreateItem)
		items.PUT("/:id", middlewares.RoleMiddleware("admin", "cashier"), controllers.UpdateItem)
		items.DELETE("/:id", middlewares.RoleMiddleware("admin", "cashier"), controllers.DeleteItem)
		items.POST("/bulk", middlewares.RoleMiddleware("admin", "cashier"), controllers.BulkCreateItems)
	}

	// Transactions
	transactions := r.Group("/transactions")
	transactions.Use(middlewares.AuthMiddleware())
	{
		transactions.POST("/", controllers.CreateTransaction)
		transactions.GET("/", controllers.GetTransactions)
		transactions.GET("/history", controllers.GetTransactionHistory)
		transactions.GET("/:id", controllers.GetTransactionByID)
		transactions.PATCH("/:id", controllers.UpdateTransactionStatus)

		transactions.POST("/:id/checkout", controllers.CheckoutTransaction)
		transactions.POST("/:id/refund", controllers.RefundTransaction)
		transactions.GET("/drafts", controllers.GetDraftTransactions)
		transactions.DELETE("/:id", controllers.DeleteTransaction)
	}

	// Dashboard
	dashboard := r.Group("/dashboard")
	dashboard.Use(middlewares.AuthMiddleware())
	{
		dashboard.GET("/", controllers.GetDashboard)
	}

	// Attendance (admin only)
	attendance := r.Group("/attendance")
	attendance.Use(middlewares.AuthMiddleware(), middlewares.RoleMiddleware("admin"))
	{
		attendance.GET("/", controllers.GetAttendances)
		attendance.POST("/", controllers.CreateAttendance)
		attendance.GET("/today", controllers.GetTodayAttendance)
		attendance.GET("/history", controllers.GetAttendanceHistory)
	}

	// Users (admin only)
	users := r.Group("/users")
	users.Use(middlewares.AuthMiddleware(), middlewares.RoleMiddleware("admin"))
	{
		users.GET("/", controllers.GetUsers)
	}
}

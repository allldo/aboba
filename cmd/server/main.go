package main

import (
	"log"
	"mango/internal/config"
	"mango/internal/handlers"
	"mango/internal/middleware"
	"mango/internal/models"

	"github.com/gin-gonic/gin"
)

func main() {
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("Ошибка подключения к БД: %v", err)
	}

	r := gin.Default()

	// Ручная настройка CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Обработчики
	userHandler := handlers.UserHandler{DB: db}
	mangaHandler := handlers.MangaHandler{DB: db}

	// Публичные маршруты
	r.POST("/api/register", userHandler.Register)
	r.POST("/api/login", userHandler.Login)

	// Публичные маршруты для манги (без авторизации)
	r.GET("/api/manga", mangaHandler.GetAllManga)
	r.GET("/api/manga/:id", mangaHandler.GetMangaByID)

	// Маршруты для всех авторизованных пользователей
	userRoutes := r.Group("/api/user")
	userRoutes.Use(middleware.AuthRequired(models.RoleUser, models.RoleAdmin, models.RoleSuperAdmin))
	{
		userRoutes.PUT("/profile", userHandler.ChangeProfile)
		userRoutes.PUT("/password", userHandler.ChangePassword)
	}

	// Маршруты для администраторов
	adminRoutes := r.Group("/api/admin")
	adminRoutes.Use(middleware.AuthRequired(models.RoleAdmin, models.RoleSuperAdmin))
	{
		// Управление пользователями
		adminRoutes.GET("/users", userHandler.GetUsers)
		adminRoutes.PUT("/users/:id/block", userHandler.BlockUser)
		adminRoutes.DELETE("/users/:id", userHandler.DeleteUser)

		// Управление мангой
		adminRoutes.GET("/manga", mangaHandler.GetAllMangaAdmin)
		adminRoutes.POST("/manga", mangaHandler.CreateManga)
		adminRoutes.PUT("/manga/:id", mangaHandler.UpdateManga)
		adminRoutes.DELETE("/manga/:id", mangaHandler.DeleteManga)
	}

	// Маршруты только для суперадминов
	superAdminRoutes := r.Group("/api/super")
	superAdminRoutes.Use(middleware.AuthRequired(models.RoleSuperAdmin))
	{
		// Специальные маршруты для суперадмина
	}

	log.Println("Сервер запущен на порту 8080")
	r.Run(":8080")
}

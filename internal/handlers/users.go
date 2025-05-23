package handlers

import (
	"database/sql"
	"mango/internal/auth"
	"mango/internal/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type UserHandler struct {
	DB *sqlx.DB
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type ChangeProfileRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// Регистрация пользователя
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, существует ли пользователь
	var exists bool
	err := h.DB.Get(&exists, "SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 OR email = $2)", req.Username, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Пользователь с таким именем или email уже существует"})
		return
	}

	// Хешируем пароль
	hashedPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка хеширования пароля"})
		return
	}

	// Создаем пользователя
	var userID int64
	err = h.DB.Get(&userID,
		"INSERT INTO users (username, email, password, role) VALUES ($1, $2, $3, $4) RETURNING id",
		req.Username, req.Email, hashedPassword, models.RoleUser)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания пользователя"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Пользователь успешно зарегистрирован",
		"user_id": userID,
	})
}

// Авторизация пользователя
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	err := h.DB.Get(&user, "SELECT id, username, email, password, role, is_blocked FROM users WHERE username = $1", req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверные учетные данные"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	// Проверяем, заблокирован ли пользователь
	if user.IsBlocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Аккаунт заблокирован"})
		return
	}

	// Проверяем пароль
	if !auth.CheckPassword(req.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверные учетные данные"})
		return
	}

	// Генерируем токен
	token, err := auth.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка генерации токена"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// Изменение профиля
func (h *UserHandler) ChangeProfile(c *gin.Context) {
	var req ChangeProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetInt64("userID")

	// Проверяем, не занято ли новое имя пользователя или email другим пользователем
	var exists bool
	err := h.DB.Get(&exists,
		"SELECT EXISTS(SELECT 1 FROM users WHERE (username = $1 OR email = $2) AND id != $3)",
		req.Username, req.Email, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Имя пользователя или email уже используется"})
		return
	}

	// Обновляем профиль
	_, err = h.DB.Exec(
		"UPDATE users SET username = $1, email = $2, updated_at = NOW() WHERE id = $3",
		req.Username, req.Email, userID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления профиля"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Профиль успешно обновлен"})
}

// Изменение пароля
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetInt64("userID")

	// Получаем текущий пароль пользователя
	var currentPassword string
	err := h.DB.Get(&currentPassword, "SELECT password FROM users WHERE id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	// Проверяем старый пароль
	if !auth.CheckPassword(req.OldPassword, currentPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный текущий пароль"})
		return
	}

	// Хешируем новый пароль
	hashedPassword, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка хеширования пароля"})
		return
	}

	// Обновляем пароль
	_, err = h.DB.Exec("UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2", hashedPassword, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления пароля"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменен"})
}

// Получение списка пользователей (только для админов)
func (h *UserHandler) GetUsers(c *gin.Context) {
	var users []models.User
	err := h.DB.Select(&users, "SELECT id, username, email, role, is_blocked, created_at, updated_at FROM users ORDER BY created_at DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения пользователей"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// Блокировка пользователя (только для админов)
func (h *UserHandler) BlockUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	// Проверяем, существует ли пользователь
	var user models.User
	err = h.DB.Get(&user, "SELECT id, role FROM users WHERE id = $1", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	// Нельзя блокировать суперадмина
	if user.Role == models.RoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Нельзя заблокировать суперадмина"})
		return
	}

	// Блокируем пользователя
	_, err = h.DB.Exec("UPDATE users SET is_blocked = true, updated_at = NOW() WHERE id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка блокировки пользователя"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь заблокирован"})
}

// Удаление пользователя (только для админов)
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID пользователя"})
		return
	}

	// Проверяем, существует ли пользователь
	var user models.User
	err = h.DB.Get(&user, "SELECT id, role FROM users WHERE id = $1", userID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Пользователь не найден"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	// Нельзя удалить суперадмина
	if user.Role == models.RoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Нельзя удалить суперадмина"})
		return
	}

	// Удаляем пользователя
	_, err = h.DB.Exec("DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления пользователя"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Пользователь удален"})
}

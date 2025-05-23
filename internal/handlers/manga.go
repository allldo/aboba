package handlers

import (
	"database/sql"
	"mango/internal/models"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type MangaHandler struct {
	DB *sqlx.DB
}

type CreateMangaRequest struct {
	Title       string             `json:"title" binding:"required,min=1,max=255"`
	Description string             `json:"description"`
	Author      string             `json:"author" binding:"required,min=1,max=255"`
	Artist      string             `json:"artist"`
	Genres      []string           `json:"genres"`
	Status      models.MangaStatus `json:"status" binding:"required"`
	Year        int                `json:"year"`
	Chapters    int                `json:"chapters"`
	Price       float64            `json:"price" binding:"required,min=0"`
	CoverImage  string             `json:"cover_image"`
	Stock       int                `json:"stock" binding:"required,min=0"`
}

type UpdateMangaRequest struct {
	Title       string             `json:"title" binding:"min=1,max=255"`
	Description string             `json:"description"`
	Author      string             `json:"author" binding:"min=1,max=255"`
	Artist      string             `json:"artist"`
	Genres      []string           `json:"genres"`
	Status      models.MangaStatus `json:"status"`
	Year        int                `json:"year"`
	Chapters    int                `json:"chapters"`
	Price       float64            `json:"price" binding:"min=0"`
	CoverImage  string             `json:"cover_image"`
	Stock       int                `json:"stock" binding:"min=0"`
	IsActive    *bool              `json:"is_active"`
}

// Получить все манги (публично доступно)
func (h *MangaHandler) GetAllManga(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	author := c.Query("author")
	status := c.Query("status")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Базовый запрос
	query := "SELECT id, title, description, author, artist, genres, status, year, chapters, price, cover_image, stock, is_active, created_at, updated_at FROM manga WHERE is_active = true"
	countQuery := "SELECT COUNT(*) FROM manga WHERE is_active = true"
	args := []interface{}{}
	argIndex := 1

	// Добавляем фильтры
	if search != "" {
		query += " AND (title ILIKE $" + strconv.Itoa(argIndex) + " OR description ILIKE $" + strconv.Itoa(argIndex) + ")"
		countQuery += " AND (title ILIKE $" + strconv.Itoa(argIndex) + " OR description ILIKE $" + strconv.Itoa(argIndex) + ")"
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if author != "" {
		query += " AND author ILIKE $" + strconv.Itoa(argIndex)
		countQuery += " AND author ILIKE $" + strconv.Itoa(argIndex)
		args = append(args, "%"+author+"%")
		argIndex++
	}

	if status != "" {
		query += " AND status = $" + strconv.Itoa(argIndex)
		countQuery += " AND status = $" + strconv.Itoa(argIndex)
		args = append(args, status)
		argIndex++
	}

	// Получаем общее количество
	var total int
	err := h.DB.Get(&total, countQuery, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подсчета манги"})
		return
	}

	// Добавляем сортировку и пагинацию
	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	var manga []models.Manga
	err = h.DB.Select(&manga, query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения манги"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"manga": manga,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + limit - 1) / limit,
		},
	})
}

// Получить мангу по ID (публично доступно)
func (h *MangaHandler) GetMangaByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID манги"})
		return
	}

	var manga models.Manga
	err = h.DB.Get(&manga,
		"SELECT id, title, description, author, artist, genres, status, year, chapters, price, cover_image, stock, is_active, created_at, updated_at FROM manga WHERE id = $1 AND is_active = true",
		id)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Манга не найдена"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения манги"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"manga": manga})
}

// Создать мангу (только админ)
func (h *MangaHandler) CreateManga(c *gin.Context) {
	var req CreateMangaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, что статус валидный
	validStatuses := []models.MangaStatus{
		models.StatusOngoing,
		models.StatusCompleted,
		models.StatusAnnounced,
		models.StatusCancelled,
	}

	statusValid := false
	for _, status := range validStatuses {
		if req.Status == status {
			statusValid = true
			break
		}
	}

	if !statusValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный статус манги"})
		return
	}

	var mangaID int64
	err := h.DB.Get(&mangaID,
		`INSERT INTO manga (title, description, author, artist, genres, status, year, chapters, price, cover_image, stock) 
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
		req.Title, req.Description, req.Author, req.Artist,
		models.StringArray(req.Genres), req.Status, req.Year,
		req.Chapters, req.Price, req.CoverImage, req.Stock)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания манги"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Манга успешно создана",
		"manga_id": mangaID,
	})
}

// Обновить мангу (только админ)
func (h *MangaHandler) UpdateManga(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID манги"})
		return
	}

	var req UpdateMangaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Проверяем, существует ли манга
	var exists bool
	err = h.DB.Get(&exists, "SELECT EXISTS(SELECT 1 FROM manga WHERE id = $1)", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Манга не найдена"})
		return
	}

	// Строим динамический запрос обновления
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.Title != "" {
		setParts = append(setParts, "title = $"+strconv.Itoa(argIndex))
		args = append(args, req.Title)
		argIndex++
	}

	if req.Description != "" {
		setParts = append(setParts, "description = $"+strconv.Itoa(argIndex))
		args = append(args, req.Description)
		argIndex++
	}

	if req.Author != "" {
		setParts = append(setParts, "author = $"+strconv.Itoa(argIndex))
		args = append(args, req.Author)
		argIndex++
	}

	if req.Artist != "" {
		setParts = append(setParts, "artist = $"+strconv.Itoa(argIndex))
		args = append(args, req.Artist)
		argIndex++
	}

	if req.Genres != nil {
		setParts = append(setParts, "genres = $"+strconv.Itoa(argIndex))
		args = append(args, models.StringArray(req.Genres))
		argIndex++
	}

	if req.Status != "" {
		setParts = append(setParts, "status = $"+strconv.Itoa(argIndex))
		args = append(args, req.Status)
		argIndex++
	}

	if req.Year != 0 {
		setParts = append(setParts, "year = $"+strconv.Itoa(argIndex))
		args = append(args, req.Year)
		argIndex++
	}

	if req.Chapters != 0 {
		setParts = append(setParts, "chapters = $"+strconv.Itoa(argIndex))
		args = append(args, req.Chapters)
		argIndex++
	}

	if req.Price != 0 {
		setParts = append(setParts, "price = $"+strconv.Itoa(argIndex))
		args = append(args, req.Price)
		argIndex++
	}

	if req.CoverImage != "" {
		setParts = append(setParts, "cover_image = $"+strconv.Itoa(argIndex))
		args = append(args, req.CoverImage)
		argIndex++
	}

	if req.Stock != 0 {
		setParts = append(setParts, "stock = $"+strconv.Itoa(argIndex))
		args = append(args, req.Stock)
		argIndex++
	}

	if req.IsActive != nil {
		setParts = append(setParts, "is_active = $"+strconv.Itoa(argIndex))
		args = append(args, *req.IsActive)
		argIndex++
	}

	if len(setParts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Нет данных для обновления"})
		return
	}

	setParts = append(setParts, "updated_at = NOW()")

	query := "UPDATE manga SET " + strings.Join(setParts, ", ") + " WHERE id = $" + strconv.Itoa(argIndex)
	args = append(args, id)

	_, err = h.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления манги"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Манга успешно обновлена"})
}

// Удалить мангу (только админ) - мягкое удаление
func (h *MangaHandler) DeleteManga(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный ID манги"})
		return
	}

	// Проверяем, существует ли манга
	var exists bool
	err = h.DB.Get(&exists, "SELECT EXISTS(SELECT 1 FROM manga WHERE id = $1)", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка базы данных"})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Манга не найдена"})
		return
	}

	// Мягкое удаление - помечаем как неактивную
	_, err = h.DB.Exec("UPDATE manga SET is_active = false, updated_at = NOW() WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка удаления манги"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Манга успешно удалена"})
}

// Получить все манги для админа (включая неактивные)
func (h *MangaHandler) GetAllMangaAdmin(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	var total int
	err := h.DB.Get(&total, "SELECT COUNT(*) FROM manga")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подсчета манги"})
		return
	}

	var manga []models.Manga
	err = h.DB.Select(&manga,
		"SELECT id, title, description, author, artist, genres, status, year, chapters, price, cover_image, stock, is_active, created_at, updated_at FROM manga ORDER BY created_at DESC LIMIT $1 OFFSET $2",
		limit, offset)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка получения манги"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"manga": manga,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + limit - 1) / limit,
		},
	})
}

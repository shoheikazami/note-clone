package main

import (
	"net/http"
	"note-clone/db"
	"note-clone/models"

	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	// --- 記事一覧 ---
	r.GET("/articles", func(c *gin.Context) {
		var articles []models.Article
		db.DB.Order("id desc").Find(&articles)
		c.HTML(http.StatusOK, "index.html", gin.H{"articles": articles})
	})

	// --- 記事作成 (バリデーション付き) ---
	r.POST("/articles", func(c *gin.Context) {
		var article models.Article
		// ここで models/article.go に書いた binding:"required" がチェックされます
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "タイトルと本文を入力してください",
				"details": err.Error(),
			})
			return
		}
		db.DB.Create(&article)
		c.JSON(http.StatusOK, article)
	})

	// --- 記事更新 (バリデーション付き) ---
	r.PUT("/articles/:id", func(c *gin.Context) {
		id := c.Param("id")
		var article models.Article
		if err := db.DB.First(&article, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "記事が見つかりません"})
			return
		}

		var input models.Article
		// 更新時も空文字を禁止するために ShouldBindJSON を使用
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "入力内容が正しくありません"})
			return
		}

		db.DB.Model(&article).Updates(input)
		c.JSON(http.StatusOK, article)
	})

	// --- 記事削除 ---
	r.DELETE("/articles/:id", func(c *gin.Context) {
		id := c.Param("id")
		db.DB.Delete(&models.Article{}, id)
		c.JSON(http.StatusOK, gin.H{"message": "削除完了"})
	})

	return r
}

func main() {
	db.Init()
	db.DB.AutoMigrate(&models.Article{})
	
	r := SetupRouter()
	r.Run(":8080")
}
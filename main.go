package main

import (
	"note-clone/db"
	"note-clone/models"
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	// 1. DBの初期化
	db.Init()
	
	// 2. テーブルの自動作成（マイグレーション）
	db.DB.AutoMigrate(&models.Article{})

	r := gin.Default()

	// 記事一覧を取得するテストAPI
	r.GET("/articles", func(c *gin.Context) {
		var articles []models.Article
		db.DB.Find(&articles)
		c.JSON(http.StatusOK, articles)
	})

	r.Run(":8080")
}
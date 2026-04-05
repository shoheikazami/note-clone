package main

import (
	"net/http"
	"note-clone/db"
	"note-clone/models"

	"github.com/gin-gonic/gin"
)

func main() {
	db.Init()
	// データベースのテーブルを最新の構造に更新
	db.DB.AutoMigrate(&models.Article{})

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	// 1. 【GET】 記事一覧を取得してHTMLを表示
	r.GET("/articles", func(c *gin.Context) {
		var articles []models.Article
		// IDの降順（新しい順）で取得するとブログらしくなります
		db.DB.Order("id desc").Find(&articles)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"articles": articles,
		})
	})

	// 2. 【POST】 新しい記事を作成
	r.POST("/articles", func(c *gin.Context) {
		var article models.Article
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		db.DB.Create(&article)
		c.JSON(http.StatusOK, article)
	})

	// 3. 【PUT】 既存の記事を更新（ここを追加！）
	r.PUT("/articles/:id", func(c *gin.Context) {
		id := c.Param("id")
		var article models.Article

		// まず、更新したい記事がDBに存在するか確認
		if err := db.DB.First(&article, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "記事が見つかりません"})
			return
		}

		// リクエストから新しいタイトルと本文を受け取る
		var input models.Article
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// DBのデータを上書き保存
		db.DB.Model(&article).Updates(input)
		c.JSON(http.StatusOK, article)
	})

	// 4. 【DELETE】 記事を削除
	r.DELETE("/articles/:id", func(c *gin.Context) {
		id := c.Param("id")
		if err := db.DB.Delete(&models.Article{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "削除に失敗しました"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "削除完了"})
	})

	r.Run(":8080")
}
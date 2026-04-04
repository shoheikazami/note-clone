package main

import (
	"net/http"
	"note-clone/db"
	"note-clone/models"

	"github.com/gin-gonic/gin"
)

func main() {
	db.Init()
	db.DB.AutoMigrate(&models.Article{})

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	// 【GET】 記事一覧を表示
	r.GET("/articles", func(c *gin.Context) {
		var articles []models.Article
		db.DB.Find(&articles)
		c.HTML(http.StatusOK, "index.html", gin.H{
			"articles": articles,
		})
	})

	// 【POST】 記事を新規作成
	r.POST("/articles", func(c *gin.Context) {
		var article models.Article
		if err := c.ShouldBindJSON(&article); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		db.DB.Create(&article)
		c.JSON(http.StatusOK, article)
	})

	// 【DELETE】 特定の記事を削除
	// URLの ":id" 部分が可変になり、どの記事を消すか指定できる
	r.DELETE("/articles/:id", func(c *gin.Context) {
		id := c.Param("id")
		
		// DBから指定されたIDの記事を削除する（論理削除）
		if err := db.DB.Delete(&models.Article{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "削除に失敗しました"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "削除完了", "id": id})
	})

	r.Run(":8080")
}
package main

import (
	"net/http"
	"note-clone/db"
	"note-clone/models"

	"github.com/gin-gonic/gin"
)

// SetupRouter はルーターの設定を定義し、テストコードからも呼び出せるようにします
func SetupRouter() *gin.Engine {
	r := gin.Default()

	// HTMLテンプレートの読み込み設定
	r.LoadHTMLGlob("templates/*")

	// --- ルート定義 ---

	// 【GET】 記事一覧を表示
	r.GET("/articles", func(c *gin.Context) {
		var articles []models.Article
		db.DB.Order("id desc").Find(&articles)
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

	// 【PUT】 記事を更新
	r.PUT("/articles/:id", func(c *gin.Context) {
		id := c.Param("id")
		var article models.Article
		if err := db.DB.First(&article, id).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "記事が見つかりません"})
			return
		}

		var input models.Article
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		db.DB.Model(&article).Updates(input)
		c.JSON(http.StatusOK, article)
	})

	// 【DELETE】 記事を削除
	r.DELETE("/articles/:id", func(c *gin.Context) {
		id := c.Param("id")
		if err := db.DB.Delete(&models.Article{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "削除に失敗しました"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "削除完了"})
	})

	return r
}

func main() {
	// データベース初期化
	db.Init()
	db.DB.AutoMigrate(&models.Article{})

	// ルーターのセットアップとサーバー起動
	r := SetupRouter()
	r.Run(":8080")
}
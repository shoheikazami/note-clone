package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"note-clone/db"
	"note-clone/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey []byte

// 認証ミドルウェア
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "トークンが必要です"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "無効なトークンです"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("username", claims["username"])
		}

		c.Next()
	}
}

func SetupRouter() *gin.Engine {
	r := gin.Default()

	// ユーザー登録
	r.POST("/signup", func(c *gin.Context) {
		var input models.User
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "入力不備があります"})
			return
		}
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		user := models.User{Username: input.Username, Password: string(hashedPassword)}
		if err := db.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ユーザー名の重複または登録失敗"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "登録完了"})
	})

	// ログイン
	r.POST("/login", func(c *gin.Context) {
		var input models.User
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "入力不備"})
			return
		}
		var user models.User
		if err := db.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "認証失敗"})
			return
		}
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "認証失敗"})
			return
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": user.Username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})
		tokenString, _ := token.SignedString(jwtKey)
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	})

	// 記事一覧取得（検索機能 + Preloadによるリレーション取得）
	r.GET("/articles", func(c *gin.Context) {
		keyword := c.Query("keyword")
		var articles []models.Article
		
		// User(投稿者) と LikedBy(いいねした人) を一括取得
		query := db.DB.Preload("User").Preload("LikedBy").Order("id desc")

		if keyword != "" {
			searchStr := "%" + keyword + "%"
			query = query.Where("title LIKE ? OR content LIKE ?", searchStr, searchStr)
		}

		if err := query.Find(&articles).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "記事の取得に失敗しました"})
			return
		}
		c.JSON(http.StatusOK, articles)
	})

	// 認証が必要なエンドポイント
	authorized := r.Group("/")
	authorized.Use(AuthMiddleware())
	{
		// 記事投稿
		authorized.POST("/articles", func(c *gin.Context) {
			var article models.Article
			if err := c.ShouldBindJSON(&article); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "タイトルと本文は必須です"})
				return
			}

			username, _ := c.Get("username")
			var user models.User
			db.DB.Where("username = ?", username).First(&user)

			article.UserID = user.ID
			if err := db.DB.Create(&article).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "保存に失敗しました"})
				return
			}
			c.JSON(http.StatusOK, article)
		})

		// いいね機能 (トグル形式)
		authorized.POST("/articles/:id/like", func(c *gin.Context) {
			articleID := c.Param("id")
			username, _ := c.Get("username")

			var user models.User
			var article models.Article

			db.DB.Where("username = ?", username).First(&user)
			if err := db.DB.First(&article, articleID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "記事が見つかりません"})
				return
			}

			// 中間テーブルをチェックし、既にいいねしていれば解除、なければ追加
			var count int64
			db.DB.Table("article_likes").Where("article_id = ? AND user_id = ?", article.ID, user.ID).Count(&count)

			if count > 0 {
				db.DB.Model(&article).Association("LikedBy").Delete(&user)
				c.JSON(http.StatusOK, gin.H{"message": "いいねを解除しました"})
			} else {
				db.DB.Model(&article).Association("LikedBy").Append(&user)
				c.JSON(http.StatusOK, gin.H{"message": "いいねしました"})
			}
		})

		// 記事削除
		authorized.DELETE("/articles/:id", func(c *gin.Context) {
			id := c.Param("id")
			username, _ := c.Get("username")
			var user models.User
			db.DB.Where("username = ?", username).First(&user)

			var article models.Article
			if err := db.DB.First(&article, id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "記事が見つかりません"})
				return
			}

			if article.UserID != user.ID {
				c.JSON(http.StatusForbidden, gin.H{"error": "他人の記事は削除できません"})
				return
			}

			db.DB.Delete(&article)
			c.JSON(http.StatusOK, gin.H{"message": "削除しました"})
		})
	}

	return r
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found, using system environment variables")
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set in environment variables")
	}
	jwtKey = []byte(secret)

	db.Init()
	// 自動マイグレーションでモデル間のリレーション（中間テーブル含む）を同期
	db.DB.AutoMigrate(&models.Article{}, &models.User{})

	r := SetupRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s...", port)
	r.Run(":" + port)
}
package main

import (
	"net/http"
	"note-clone/db"
	"note-clone/models"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtKey = []byte("your_secret_key")

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
	r.LoadHTMLGlob("templates/*")

	// --- 認証不要ルート ---
	r.POST("/signup", func(c *gin.Context) {
		var input models.User
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "入力不備"})
			return
		}
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		user := models.User{Username: input.Username, Password: string(hashedPassword)}
		if err := db.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "登録失敗"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "登録完了"})
	})

	r.POST("/login", func(c *gin.Context) {
		var input models.User
		c.ShouldBindJSON(&input)
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

	r.GET("/articles", func(c *gin.Context) {
		var articles []models.Article
		db.DB.Order("id desc").Find(&articles)
		c.HTML(http.StatusOK, "index.html", gin.H{"articles": articles})
	})

	// --- 認証が必要なルート ---
	authorized := r.Group("/")
	authorized.Use(AuthMiddleware())
	{
		// 記事投稿
		authorized.POST("/articles", func(c *gin.Context) {
			var article models.Article
			if err := c.ShouldBindJSON(&article); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "入力不備"})
				return
			}
			username, _ := c.Get("username")
			var user models.User
			db.DB.Where("username = ?", username).First(&user)

			article.UserID = user.ID
			db.DB.Create(&article)
			c.JSON(http.StatusOK, article)
		})

		// 記事削除（認可チェック付き）
		authorized.DELETE("/articles/:id", func(c *gin.Context) {
			id := c.Param("id")
			username, _ := c.Get("username")
			
			// 1. 実行ユーザーの特定
			var user models.User
			db.DB.Where("username = ?", username).First(&user)

			// 2. 記事の存在確認
			var article models.Article
			if err := db.DB.First(&article, id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "記事が見つかりません"})
				return
			}

			// 3. 所有権の検証 (Authorization)
			if article.UserID != user.ID {
				c.JSON(http.StatusForbidden, gin.H{"error": "他人の記事は操作できません"})
				return
			}

			db.DB.Delete(&article)
			c.JSON(http.StatusOK, gin.H{"message": "削除しました"})
		})

		// 記事更新（認可チェック付き）
		authorized.PUT("/articles/:id", func(c *gin.Context) {
			id := c.Param("id")
			username, _ := c.Get("username")
			
			var user models.User
			db.DB.Where("username = ?", username).First(&user)

			var article models.Article
			if err := db.DB.First(&article, id).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "記事が見つかりません"})
				return
			}

			// 所有権の検証
			if article.UserID != user.ID {
				c.JSON(http.StatusForbidden, gin.H{"error": "他人の記事は操作できません"})
				return
			}

			// 入力バリデーション
			var input models.Article
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "入力不備"})
				return
			}

			db.DB.Model(&article).Updates(input)
			c.JSON(http.StatusOK, article)
		})
	}

	return r
}

func main() {
	db.Init()
	db.DB.AutoMigrate(&models.Article{}, &models.User{})
	r := SetupRouter()
	r.Run(":8080")
}
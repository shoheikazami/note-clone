package main

import (
	"log"
	"net/http"
	"note-clone/db"
	"note-clone/models"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

// jwtKeyをグローバル変数として定義（mainで環境変数から値を代入）
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
	
	// 認証不要ルート
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
		c.JSON(http.StatusOK, articles)
	})

	// 認証が必要なルート
	authorized := r.Group("/")
	authorized.Use(AuthMiddleware())
	{
		authorized.POST("/articles", func(c *gin.Context) {
			var article models.Article
			c.ShouldBindJSON(&article)
			username, _ := c.Get("username")
			var user models.User
			db.DB.Where("username = ?", username).First(&user)
			article.UserID = user.ID
			db.DB.Create(&article)
			c.JSON(http.StatusOK, article)
		})

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
				c.JSON(http.StatusForbidden, gin.H{"error": "他人の記事は操作できません"})
				return
			}
			db.DB.Delete(&article)
			c.JSON(http.StatusOK, gin.H{"message": "削除しました"})
		})
	}

	return r
}

func main() {
	// 1. 環境変数の読み込み
	err := godotenv.Load()
	if err != nil {
		log.Println(".envファイルが見つかりません。システム環境変数を使用します。")
	}

	// 2. JWT秘密鍵の設定
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRETが設定されていません")
	}
	jwtKey = []byte(secret)

	// 3. DB初期化
	db.Init()
	db.DB.AutoMigrate(&models.Article{}, &models.User{})

	// 4. ルーターセットアップと起動
	r := SetupRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	r.Run(":" + port)
}
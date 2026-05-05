package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// --- Models ---

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"unique;not null" json:"username"`
	Password  string    `json:"-"` // パスワードはJSONに含めない
	CreatedAt time.Time `json:"created_at"`
}

type Article struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	ImageURL  string    `json:"image_url"`
	UserID    uint      `json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user"`
	LikedBy   []User    `gorm:"many2many:article_likes;" json:"liked_by"`
	CreatedAt time.Time `json:"created_at"`
}

// --- JWT Claims ---

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

var jwtKey = []byte("your_secret_key")

// --- Database ---

var db *gorm.DB

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		panic("DATABASE_URL is not set")
	}

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// ユーザー、記事、および中間テーブルのマイグレーション
	db.AutoMigrate(&User{}, &Article{})
}

// --- Middleware ---

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Next()
	}
}

// --- Main ---

func main() {
	initDB()

	r := gin.Default()

	// 疎通確認用
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Note Clone API is running on PostgreSQL!"})
	})

	// ユーザー登録
	r.POST("/signup", func(c *gin.Context) {
		var user User
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		user.Password = string(hashedPassword)
		db.Create(&user)
		c.JSON(http.StatusOK, gin.H{"message": "Signup successful"})
	})

	// ログイン
	r.POST("/login", func(c *gin.Context) {
		var input User
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var user User
		if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
			return
		}

		expirationTime := time.Now().Add(24 * time.Hour)
		claims := &Claims{
			UserID: user.ID,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expirationTime.Unix(),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(jwtKey)

		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	})

	// 記事取得（キーワード検索対応）
	r.GET("/articles", func(c *gin.Context) {
		var articles []Article
		query := c.Query("q")
		
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
		offset := (page - 1) * limit

		dbQuery := db.Preload("User").Preload("LikedBy")

		if query != "" {
			// タイトルまたは内容から部分一致検索
			dbQuery = dbQuery.Where("title ILIKE ? OR content ILIKE ?", "%"+query+"%", "%"+query+"%")
		}

		dbQuery.Order("created_at desc").Offset(offset).Limit(limit).Find(&articles)
		c.JSON(http.StatusOK, articles)
	})

	// 認証が必要なルート
	auth := r.Group("/")
	auth.Use(AuthMiddleware())
	{
		// 記事投稿
		auth.POST("/articles", func(c *gin.Context) {
			var article Article
			if err := c.ShouldBindJSON(&article); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			article.UserID = c.MustGet("userID").(uint)
			db.Create(&article)
			c.JSON(http.StatusOK, article)
		})

		// いいね機能
		auth.POST("/articles/:id/like", func(c *gin.Context) {
			articleID := c.Param("id")
			userID := c.MustGet("userID").(uint)

			var article Article
			if err := db.First(&article, articleID).Error; err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Article not found"})
				return
			}

			var user User
			db.First(&user, userID)

			// 中間テーブルへ保存
			db.Model(&article).Association("LikedBy").Append(&user)
			c.JSON(http.StatusOK, gin.H{"message": "Liked successfully"})
		})
	}

	// Render用のポート設定
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
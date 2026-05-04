package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// --- Models ---
type User struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `gorm:"unique;not null" json:"username"`
	Password string `gorm:"not null" json:"-"`
}

type Article struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Title     string    `gorm:"not null" json:"title"`
	Content   string    `gorm:"not null" json:"content"`
	ImageURL  string    `json:"image_url"` 
	UserID    uint      `json:"user_id"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	LikedBy   []User    `json:"liked_by" gorm:"many2many:article_likes;"`
	CreatedAt time.Time `json:"created_at"`
}

var db *gorm.DB

// --- Utils ---
func uploadToCloudinary(file interface{}) (string, error) {
	ctx := context.Background()
	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		return "", err
	}

	uploadResult, err := cld.Upload.Upload(ctx, file, uploader.UploadParams{})
	if err != nil {
		return "", err
	}

	return uploadResult.SecureURL, nil
}

// --- Middleware ---
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "トークンが必要です"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "無効なトークンです"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// float64からuintへ変換してセット
			c.Set("userID", uint(claims["user_id"].(float64)))
			c.Next()
		}
	}
}

func main() {
	_ = godotenv.Load()

	var err error
	db, err = gorm.Open(sqlite.Open("note.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}
	db.AutoMigrate(&User{}, &Article{})

	r := gin.Default()

	// --- Routes ---

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Note Clone API is running!"})
	})

	r.POST("/signup", func(c *gin.Context) {
		var input struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "入力不備があります"})
			return
		}
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 10)
		user := User{Username: input.Username, Password: string(hashedPassword)}
		if err := db.Create(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "登録に失敗しました"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "登録完了"})
	})

	r.POST("/login", func(c *gin.Context) {
		var input struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "入力不備があります"})
			return
		}

		var user User
		if err := db.Where("username = ?", input.Username).First(&user).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "認証に失敗しました"})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "認証に失敗しました"})
			return
		}

		// ペイロードにuser_idを含めるよう修正
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": user.ID,
			"exp":     time.Now().Add(time.Hour * 24).Unix(),
		})

		tokenString, _ := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	})

	r.GET("/articles", func(c *gin.Context) {
		var articles []Article
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
		offset := (page - 1) * limit

		db.Preload("User").Preload("LikedBy").Offset(offset).Limit(limit).Find(&articles)
		c.JSON(http.StatusOK, articles)
	})

	auth := r.Group("/")
	auth.Use(AuthMiddleware())
	{
		auth.POST("/articles", func(c *gin.Context) {
			title := c.PostForm("title")
			content := c.PostForm("content")
			userID := c.MustGet("userID").(uint)

			file, _, err := c.Request.FormFile("image")
			var imageURL string
			if err == nil {
				url, uploadErr := uploadToCloudinary(file)
				if uploadErr == nil {
					imageURL = url
				}
			}

			article := Article{
				Title:    title,
				Content:  content,
				ImageURL: imageURL,
				UserID:   userID,
			}
			db.Create(&article)
			c.JSON(http.StatusOK, article)
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
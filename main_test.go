package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"note-clone/db"
	"note-clone/models"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func setupTest() *gin.Engine {
	dialector := sqlite.Open(":memory:")
	db.DB, _ = gorm.Open(dialector, &gorm.Config{})
	db.DB.AutoMigrate(&models.Article{}, &models.User{})
	gin.SetMode(gin.TestMode)
	return SetupRouter()
}

func TestCreateArticleWithUserBinding(t *testing.T) {
	r := setupTest()

	// 1. テストユーザーの作成とログイン
	username := "test_author"
	user := models.User{Username: username, Password: "password123"}
	jsonUser, _ := json.Marshal(user)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/signup", bytes.NewBuffer(jsonUser)))

	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonUser)))
	
	var loginRes map[string]string
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)
	token := loginRes["token"]

	// DBから作成されたユーザーのIDを取得（期待値として使用）
	var createdUser models.User
	db.DB.Where("username = ?", username).First(&createdUser)

	// 2. 認証トークンを使って記事を投稿
	articlePayload := models.Article{Title: "紐付けテスト", Content: "ユーザーIDが入るはず"}
	jsonArt, _ := json.Marshal(articlePayload)
	req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonArt))
	req.Header.Set("Authorization", "Bearer "+token)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 3. アサーション（検証）
	assert.Equal(t, http.StatusOK, w.Code)

	var createdArticle models.Article
	json.Unmarshal(w.Body.Bytes(), &createdArticle)

	// レスポンスの user_id が、ログインしたユーザーの ID と一致するか
	assert.Equal(t, createdUser.ID, createdArticle.UserID)
	assert.Equal(t, "紐付けテスト", createdArticle.Title)

	// DB側でもう一度確認（念のため）
	var dbArticle models.Article
	db.DB.First(&dbArticle, createdArticle.ID)
	assert.Equal(t, createdUser.ID, dbArticle.UserID)
}
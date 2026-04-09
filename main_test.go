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

// 認証が必要な記事投稿のテスト
func TestCreateArticleWithAuth(t *testing.T) {
	r := setupTest()

	// 1. ユーザー作成とログインしてトークン取得
	user := models.User{Username: "author", Password: "password"}
	jsonUser, _ := json.Marshal(user)
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/signup", bytes.NewBuffer(jsonUser)))

	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonUser)))
	
	var loginRes map[string]string
	json.Unmarshal(wLogin.Body.Bytes(), &loginRes)
	token := loginRes["token"]

	// 2. トークンなしで投稿（拒否されるはず）
	reqNoToken, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer([]byte(`{"title":"NoToken"}`)))
	wNoToken := httptest.NewRecorder()
	r.ServeHTTP(wNoToken, reqNoToken)
	assert.Equal(t, http.StatusUnauthorized, wNoToken.Code)

	// 3. トークンありで投稿（成功するはず）
	article := models.Article{Title: "AuthTitle", Content: "AuthContent"}
	jsonArt, _ := json.Marshal(article)
	reqWithToken, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonArt))
	reqWithToken.Header.Set("Authorization", "Bearer "+token) // トークンをセット
	wWithToken := httptest.NewRecorder()
	r.ServeHTTP(wWithToken, reqWithToken)

	assert.Equal(t, http.StatusOK, wWithToken.Code)
}
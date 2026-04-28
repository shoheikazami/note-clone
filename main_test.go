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

// テスト用のセットアップ（メモリ内DBを使用）
func setupTest() *gin.Engine {
	dialector := sqlite.Open(":memory:")
	db.DB, _ = gorm.Open(dialector, &gorm.Config{})
	db.DB.AutoMigrate(&models.Article{}, &models.User{})
	gin.SetMode(gin.TestMode)
	return SetupRouter()
}

// ユーザー作成とログインをまとめてトークンを返すヘルパー
func loginUser(t *testing.T, r *gin.Engine, username string) string {
	user := models.User{Username: username, Password: "password"}
	jsonUser, _ := json.Marshal(user)
	
	// Signup
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/signup", bytes.NewBuffer(jsonUser)))
	
	// Login
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/login", bytes.NewBuffer(jsonUser)))
	
	var res map[string]string
	json.Unmarshal(w.Body.Bytes(), &res)
	return res["token"]
}

func TestDeleteArticleAuthorization(t *testing.T) {
	r := setupTest()

	// 1. ユーザーAでログインし、記事を作成
	tokenA := loginUser(t, r, "userA")
	articlePayload := models.Article{Title: "Aさんの秘密", Content: "これはAさんの記事です"}
	jsonArt, _ := json.Marshal(articlePayload)
	
	reqCreate, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonArt))
	reqCreate.Header.Set("Authorization", "Bearer "+tokenA)
	wCreate := httptest.NewRecorder()
	r.ServeHTTP(wCreate, reqCreate)
	
	var createdArticle models.Article
	json.Unmarshal(wCreate.Body.Bytes(), &createdArticle)

	// 2. ユーザーBでログイン
	tokenB := loginUser(t, r, "userB")

	// 3. ユーザーBのトークンを使って、ユーザーAの記事(ID: 1)を削除しようとする
	reqDelete, _ := http.NewRequest("DELETE", "/articles/1", nil)
	reqDelete.Header.Set("Authorization", "Bearer "+tokenB)
	wDelete := httptest.NewRecorder()
	r.ServeHTTP(wDelete, reqDelete)

	// 4. 検証：403 Forbidden であること
	assert.Equal(t, http.StatusForbidden, wDelete.Code)
	
	// 5. DBに記事がまだ残っていることを確認
	var dbArticle models.Article
	result := db.DB.First(&dbArticle, createdArticle.ID)
	assert.Nil(t, result.Error) // エラーがない ＝ 記事が消されずに残っている
}
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
	var err error
	db.DB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		panic("テスト用DBの接続に失敗しました: " + err.Error())
	}

	db.DB.AutoMigrate(&models.Article{})
	gin.SetMode(gin.TestMode)
	return SetupRouter()
}

// 正常な投稿のテスト
func TestCreateArticle(t *testing.T) {
	r := setupTest()

	article := models.Article{Title: "正常な記事", Content: "正常な本文"}
	jsonValue, _ := json.Marshal(article)

	req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// 【追加】バリデーションエラーのテスト
func TestCreateArticleValidationError(t *testing.T) {
	r := setupTest()

	// タイトルが空のデータ（binding:"required" で弾かれるべきもの）
	invalidArticle := models.Article{Title: "", Content: "本文はあるけどタイトルがない"}
	jsonValue, _ := json.Marshal(invalidArticle)

	req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)

	// ステータスコードが 400 Bad Request であることを期待する
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	// エラーメッセージが含まれているか確認
	assert.Contains(t, w.Body.String(), "error")
}

func TestDeleteArticle(t *testing.T) {
	r := setupTest()
	target := models.Article{Title: "削除用", Content: "消えます"}
	db.DB.Create(&target)

	req, _ := http.NewRequest("DELETE", "/articles/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateArticle(t *testing.T) {
	r := setupTest()
	db.DB.Create(&models.Article{Title: "旧", Content: "旧"})

	update := models.Article{Title: "新", Content: "新"}
	jsonValue, _ := json.Marshal(update)

	req, _ := http.NewRequest("PUT", "/articles/1", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
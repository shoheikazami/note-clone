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
	"github.com/glebarez/sqlite" // gorm.io/driver/sqlite からこちらに変更
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// setupTest は各テストの前に、クリーンなメモリ内DB環境を構築します
func setupTest() *gin.Engine {
	// CGOを必要としない Pure Go 版の SQLite ドライバでメモリDBを開く
	dialector := sqlite.Open(":memory:")
	
	var err error
	db.DB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		panic("テスト用DBの接続に失敗しました: " + err.Error())
	}

	// データベースのテーブルを作成
	db.DB.AutoMigrate(&models.Article{})

	// Ginをテストモードに設定
	gin.SetMode(gin.TestMode)

	// main.go で定義した SetupRouter を呼び出す
	return SetupRouter()
}

// 1. 記事作成のテスト
func TestCreateArticle(t *testing.T) {
	r := setupTest()

	// 準備
	article := models.Article{Title: "テスト記事", Content: "テスト本文"}
	jsonValue, _ := json.Marshal(article)

	// 実行
	req, _ := http.NewRequest("POST", "/articles", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 検証
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response models.Article
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "テスト記事", response.Title)
}

// 2. 記事削除のテスト
func TestDeleteArticle(t *testing.T) {
	r := setupTest()

	// 準備: あらかじめデータを投入
	target := models.Article{Title: "消える記事", Content: "さらば"}
	db.DB.Create(&target)

	// 実行: ID 1番を削除
	req, _ := http.NewRequest("DELETE", "/articles/1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 検証
	assert.Equal(t, http.StatusOK, w.Code)

	var result models.Article
	err := db.DB.First(&result, 1).Error
	assert.Error(t, err) // 見つからないことが正しい
}

// 3. 記事更新のテスト
func TestUpdateArticle(t *testing.T) {
	r := setupTest()

	// 準備: 元データ
	db.DB.Create(&models.Article{Title: "旧タイトル", Content: "旧内容"})

	// 更新データ
	update := models.Article{Title: "新タイトル", Content: "新内容"}
	jsonValue, _ := json.Marshal(update)

	// 実行
	req, _ := http.NewRequest("PUT", "/articles/1", bytes.NewBuffer(jsonValue))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 検証
	assert.Equal(t, http.StatusOK, w.Code)
	
	var updated models.Article
	db.DB.First(&updated, 1)
	assert.Equal(t, "新タイトル", updated.Title)
}
package models

import "gorm.io/gorm"

type Article struct {
	gorm.Model
	// binding:"required" を追加して、空の送信をバリデーションエラーにする
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
}
package models

import "gorm.io/gorm"

// Article 記事の情報を管理する構造体
type Article struct {
	gorm.Model
	Title   string `json:"title" binding:"required"`   // バリデーション追加
	Content string `json:"content" binding:"required"` // バリデーション追加
	
	// Userとのリレーション設定
	UserID  uint   `json:"user_id"`
	User    User   `json:"user" gorm:"foreignKey:UserID"` // Preload用
}
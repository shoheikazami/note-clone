package models

import "gorm.io/gorm"

// Article 記事の情報を管理する構造体
type Article struct {
	gorm.Model
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	
	// 投稿者とのリレーション (Many-to-One)
	UserID  uint   `json:"user_id"`
	User    User   `json:"user" gorm:"foreignKey:UserID"`

	// いいね機能のリレーション (Many-to-Many)
	// gorm:"many2many:article_likes;" により、中間テーブルが自動生成されます
	LikedBy []User `json:"liked_by" gorm:"many2many:article_likes;"`
}
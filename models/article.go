package models

import "gorm.io/gorm"

// Article 記事の情報を管理する構造体
type Article struct {
    ID        uint      `gorm:"primaryKey" json:"id"`
    Title     string    `gorm:"not null" json:"title" binding:"required"`
    Content   string    `gorm:"not null" json:"content" binding:"required"`
    ImageURL  string    `json:"image_url"` // 追加：画像のURLを保存
    UserID    uint      `json:"user_id"`
    User      User      `json:"user" gorm:"foreignKey:UserID"`
    LikedBy   []User    `json:"liked_by" gorm:"many2many:article_likes;"`
    CreatedAt time.Time `json:"created_at"`
}
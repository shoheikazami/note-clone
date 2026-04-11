package models

import "gorm.io/gorm"

type Article struct {
	gorm.Model
	Title   string `json:"title" binding:"required"`
	Content string `json:"content" binding:"required"`
	
	// UserID を追加（これが外部キーになります）
	// json:"user_id" とすることで、レスポンスにも誰の投稿か含まれるようになります
	UserID  uint   `json:"user_id"`
	
	// 必要に応じて User 構造体自体を紐付ける「リレーション」の定義も可能です
	// User User `gorm:"foreignKey:UserID" json:"-"`
}
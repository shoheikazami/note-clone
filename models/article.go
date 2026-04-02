package models

import "gorm.io/gorm"

type Article struct {
	gorm.Model        // ID, CreatedAt, UpdatedAt, DeletedAt を自動追加
	Title   string `json:"title"`
	Content string `json:"content"`
}
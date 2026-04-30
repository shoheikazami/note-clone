Go (Gin) を使用した、バックエンド特化型の note クローンプロジェクトです。

## 技術スタック
- **Language**: Go 1.2x
- **Framework**: Gin
- **ORM**: GORM
- **Database**: SQLite
- **Auth**: JWT (v5), bcrypt

## 特徴
- **認可 (Authorization)**: `UserID` を用いた所有権チェックを行い、不正な操作（他人の記事の削除・編集）を 403 Forbidden で防ぎます。
- **自動テスト**: 全ての主要エンドポイントに対し、正常系・異常系のテストを完備しています。

## 開発の始め方
```bash
git clone https://github.com/shoheikazami/note-clone
go mod download
go run main.go
```

## テスト
```bash
go test -v
```

## 今後の展望
・ 環境変数の導入 (.env) による秘密鍵の管理

・ go-playground/validator を用いた入力バリデーションの強化

・ GORM Preload を用いた、記事一覧への投稿者名表示

・ 画像アップロード機能（アイキャッチ画像対応）

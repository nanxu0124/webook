package domain

import "time"

type Article struct {
	Id      int64
	Title   string
	Content string
	// 作者
	Author Author
	Status ArticleStatus

	Ctime time.Time
	Utime time.Time
}

// Author 在帖子这个领域内，
// 没有用户的概念，只有作者的概念
type Author struct {
	Id   int64
	Name string
}

// Abstract 取部分作为摘要
func (a Article) Abstract() string {
	cs := []rune(a.Content)
	if len(cs) < 100 {
		return a.Content
	}
	return string(cs[:100])
}

type ArticleStatus uint8

func (s ArticleStatus) ToUint8() uint8 {
	return uint8(s)
}

const (
	// ArticleStatusUnknown 未知状态
	ArticleStatusUnknown ArticleStatus = iota
	// ArticleStatusUnpublished 未发表
	ArticleStatusUnpublished
	// ArticleStatusPublished 已发表
	ArticleStatusPublished
	// ArticleStatusPrivate 仅自己可见
	ArticleStatusPrivate
)

package domain

type Article struct {
	Id      int64
	Title   string
	Content string
	// 作者
	Author Author
	Status ArticleStatus
}

// Author 在帖子这个领域内，
// 没有用户的概念，只有作者的概念
type Author struct {
	Id int64
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

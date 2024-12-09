package domain

type Article struct {
	Id      int64
	Title   string
	Content string
	// 作者
	Author Author
}

// Author 在帖子这个领域内，
// 没有用户的概念，只有作者的概念
type Author struct {
	Id int64
}

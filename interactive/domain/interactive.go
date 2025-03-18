package domain

// Interactive 这个是总体交互的计数
type Interactive struct {
	Biz        string `json:"biz"`
	BizId      int64  `json:"biz_id"`
	ReadCnt    int64  `json:"read_cnt"`
	LikeCnt    int64  `json:"like_cnt"`
	CollectCnt int64  `json:"collect_cnt"`
	// 这个是当下这个资源，有没有点赞或者收藏
	Liked     bool `json:"liked"`
	Collected bool `json:"collected"`
}

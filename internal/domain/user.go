package domain

type User struct {
	Id         int64  `json:"id"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	Phone      string `json:"phone"`
	WechatInfo WechatInfo

	Nickname        string `json:"nickname"`
	BirthDay        string `json:"birthDay"`
	PersonalProfile string `json:"personalProfile"`
}

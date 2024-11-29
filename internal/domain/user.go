package domain

import "time"

type User struct {
	Email    string
	Password string
	Ctime    time.Time
}

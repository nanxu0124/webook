package repository

import (
	"context"
	"database/sql"
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"webook/internal/domain"
	"webook/internal/repository/dao"
	daomocks "webook/internal/repository/dao/mocks"
)

func TestMySQLUserRepository_FindByEmail(t *testing.T) {
	testCases := []struct {
		name string
		mock func(ctrl *gomock.Controller) dao.UserDAO

		email string

		wantUser domain.User
		wantErr  error
	}{
		{
			name: "查找成功",
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().FindByEmail(gomock.Any(), "123@qq.com").
					Return(dao.User{
						Id: 1,
						Email: sql.NullString{
							String: "123@qq.com",
							Valid:  true,
						},
						Password: "hello#world123",
						Phone: sql.NullString{
							String: "15110408888",
							Valid:  true,
						},
						Ctime:           123456,
						Utime:           123456,
						Nickname:        "nanxu",
						BirthDay:        "19990124",
						PersonalProfile: "this is a personal profile",
					}, nil)
				return d
			},
			email: "123@qq.com",

			wantUser: domain.User{
				Id:              1,
				Email:           "123@qq.com",
				Password:        "hello#world123",
				Phone:           "15110408888",
				Nickname:        "nanxu",
				BirthDay:        "19990124",
				PersonalProfile: "this is a personal profile",
			},
			wantErr: nil,
		},
		{
			name: "查找失败",
			mock: func(ctrl *gomock.Controller) dao.UserDAO {
				d := daomocks.NewMockUserDAO(ctrl)
				d.EXPECT().FindByEmail(gomock.Any(), "123@qq.com").
					Return(dao.User{}, errors.New("this is error"))
				return d
			},
			email: "123@qq.com",

			wantUser: domain.User{},
			wantErr:  errors.New("this is error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			userdao := NewUserRepository(tc.mock(ctrl), nil)
			u, err := userdao.FindByEmail(context.Background(), tc.email)

			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantUser, u)
		})
	}
}

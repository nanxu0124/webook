package dao

import (
	"testing"
)

func TestGormUserDAO_Insert(t *testing.T) {
	//testCases := []struct {
	//	name string
	//
	//	mock func(t *testing.T) *sql.DB
	//
	//	ctx  context.Context
	//	user User
	//
	//	wantErr error
	//	wantId  int64
	//}{
	//	{
	//		name: "插入成功",
	//		mock: func(t *testing.T) *sql.DB {
	//			mockDB, mock, err := sqlmock.New()
	//
	//			res := sqlmock.NewResult(3, 1)
	//			mock.ExpectExec("INSERT INTO `users` .*").WillReturnError(res)
	//			require.NoError(t, err)
	//
	//			return mockDB
	//		},
	//
	//		ctx: context.Background(),
	//		user: User{
	//			Id: 3,
	//		},
	//
	//		wantErr: nil,
	//	},
	//}
	//
	//for _, tc := range testCases {
	//	t.Run(tc.name, func(t *testing.T) {
	//		db, err := gorm.Open(mysql.New(mysql.Config{
	//			Conn:                      tc.mock(t),
	//			SkipInitializeWithVersion: true,
	//		}),
	//			&gorm.Config{
	//				DisableAutomaticPing:   true,
	//				SkipDefaultTransaction: true,
	//			})
	//
	//		d := NewUserDAO(db)
	//
	//		err = d.Insert(tc.ctx, tc.user)
	//		assert.Equal(t, tc.wantErr, err)
	//		assert.Equal(t, tc.wantId, err)
	//	})
	//}
}

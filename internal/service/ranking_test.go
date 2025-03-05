package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
	"time"
	"webook/internal/domain"
	svcmocks "webook/internal/service/mocks"
)

func TestBatchRankingService_rankTopN(t *testing.T) {
	const batchSize = 2
	now := time.Now()
	testCases := []struct {
		name string
		mock func(ctrl *gomock.Controller) (InteractiveService,
			ArticleService)
		wantErr error
		wantRes []domain.Article
	}{
		{
			name: "计算成功-两批次",
			mock: func(ctrl *gomock.Controller) (InteractiveService, ArticleService) {
				intrSvc := svcmocks.NewMockInteractiveService(ctrl)
				artSvc := svcmocks.NewMockArticleService(ctrl)
				artSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 0, batchSize).
					Return([]domain.Article{
						{Id: 1, Utime: now},
						{Id: 2, Utime: now},
					}, nil)
				artSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 2, batchSize).
					Return([]domain.Article{
						{Id: 4, Utime: now},
						{Id: 3, Utime: now},
					}, nil)
				artSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 4, batchSize).
					Return([]domain.Article{}, nil)
				intrSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{1, 2}).
					Return(map[int64]domain.Interactive{
						1: {LikeCnt: 1},
						2: {LikeCnt: 2},
					}, nil)
				intrSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{4, 3}).
					Return(map[int64]domain.Interactive{
						3: {LikeCnt: 3},
						4: {LikeCnt: 4},
					}, nil)
				intrSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{}).
					Return(map[int64]domain.Interactive{}, nil)
				return intrSvc, artSvc
			},
			wantRes: []domain.Article{
				{Id: 4, Utime: now},
				{Id: 3, Utime: now},
				{Id: 2, Utime: now},
			},
		},
		{
			name: "intr失败",
			mock: func(ctrl *gomock.Controller) (InteractiveService, ArticleService) {
				intrSvc := svcmocks.NewMockInteractiveService(ctrl)
				artSvc := svcmocks.NewMockArticleService(ctrl)
				artSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 0, batchSize).
					Return([]domain.Article{
						{Id: 1, Utime: time.Now()},
						{Id: 2, Utime: time.Now()},
					}, nil)
				artSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 2, batchSize).
					Return([]domain.Article{
						{Id: 4, Utime: time.Now()},
						{Id: 3, Utime: time.Now()},
					}, nil)
				intrSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{1, 2}).
					Return(map[int64]domain.Interactive{
						1: {LikeCnt: 1},
						2: {LikeCnt: 2},
					}, nil)
				intrSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{4, 3}).
					Return(nil, errors.New("mock intr error"))
				return intrSvc, artSvc
			},
			wantErr: errors.New("mock intr error"),
		},
		{
			name: "art失败",
			mock: func(ctrl *gomock.Controller) (InteractiveService, ArticleService) {
				intrSvc := svcmocks.NewMockInteractiveService(ctrl)
				artSvc := svcmocks.NewMockArticleService(ctrl)
				artSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 0, batchSize).
					Return([]domain.Article{
						{Id: 1, Utime: time.Now()},
						{Id: 2, Utime: time.Now()},
					}, nil)
				artSvc.EXPECT().ListPub(gomock.Any(), gomock.Any(), 2, batchSize).
					Return([]domain.Article{
						{Id: 4, Utime: time.Now()},
						{Id: 3, Utime: time.Now()},
					}, errors.New("mock art error"))

				intrSvc.EXPECT().GetByIds(gomock.Any(), "article", []int64{1, 2}).
					Return(map[int64]domain.Interactive{
						1: {LikeCnt: 1},
						2: {LikeCnt: 2},
					}, nil)
				return intrSvc, artSvc
			},
			wantErr: errors.New("mock art error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			intrSvc, artSvc := tc.mock(ctrl)
			svc := &BatchRankingService{
				intrSvc:   intrSvc,
				artSvc:    artSvc,
				BatchSize: batchSize,
				N:         3,
				scoreFunc: func(likeCnt int64, utime time.Time) float64 {
					return float64(likeCnt)
				},
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			res, err := svc.rankTopN(ctx)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantRes, res)
		})
	}
}

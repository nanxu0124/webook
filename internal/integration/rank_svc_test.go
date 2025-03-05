package integration

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"testing"
	"time"
	"webook/internal/domain"
	"webook/internal/integration/startup"
	"webook/internal/repository/dao"
	"webook/internal/repository/dao/article"
	"webook/internal/service"
)

type RankingServiceTestSuite struct {
	suite.Suite
	db  *gorm.DB
	rdb redis.Cmdable
}

func TestRankService(t *testing.T) {
	suite.Run(t, &RankingServiceTestSuite{})
}

func (r *RankingServiceTestSuite) SetupSuite() {
	r.rdb = startup.InitTestRedis()
	r.db = startup.InitTestDB()
}

func (r *RankingServiceTestSuite) TearDownTest() {
	err := r.db.Exec("TRUNCATE TABLE `interactives`").Error
	require.NoError(r.T(), err)
	err = r.db.Exec("TRUNCATE TABLE `published_articles`").Error
	require.NoError(r.T(), err)
}

func (r *RankingServiceTestSuite) TestRankTopN() {
	t := r.T()
	// 设置一分钟过期时间
	svc := startup.InitRankingService(time.Minute).(*service.BatchRankingService)
	svc.BatchSize = 10
	svc.N = 10

	rdb := startup.InitTestRedis()
	db := startup.InitTestDB()
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)

		wantErr error
	}{
		{
			name: "计算成功",
			before: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
				defer cancel()
				// id 小的，点赞数多，并且发表时间比较晚
				now := time.Now()
				db = db.WithContext(ctx)
				// 准备一百条数据
				for i := 0; i < 100; i++ {
					err := db.Create(&dao.Interactive{
						BizId:   int64(i + 1),
						Biz:     "article",
						LikeCnt: int64(1000 - i*10),
					}).Error
					require.NoError(t, err)
					err = db.Create(article.PublishedArticle{
						Article: article.Article{
							Id:    int64(i + 1),
							Utime: now.Add(-time.Duration(i) * time.Hour).Unix(),
						},
					}).Error
					require.NoError(t, err)
				}
			},
			after: func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
				defer cancel()

				vals, err := rdb.Get(ctx, "ranking:article").Bytes()
				require.NoError(t, err)
				var data []domain.Article
				err = json.Unmarshal(vals, &data)
				require.NoError(t, err)
				assert.Equal(t,
					[]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
					[]int64{data[0].Id, data[1].Id, data[2].Id, data[3].Id, data[4].Id,
						data[5].Id, data[6].Id, data[7].Id, data[8].Id, data[9].Id})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			defer tc.after(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
			defer cancel()
			err := svc.RankTopN(ctx)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

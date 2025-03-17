package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/IBM/sarama"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"testing"
	"webook/interactive/repository/dao"
	"webook/internal/domain"
	"webook/internal/integration/startup"
	"webook/internal/repository/dao/article"
	"webook/internal/web"
	ijwt "webook/internal/web/jwt"
)

type ArticleGORMHandlerTestSuite struct {
	suite.Suite
	server      *gin.Engine
	db          *gorm.DB
	kafkaClient sarama.Client
}

func (s *ArticleGORMHandlerTestSuite) SetupSuite() {
	s.server = gin.Default()
	s.server.Use(func(context *gin.Context) {
		// 直接设置好
		context.Set("user", ijwt.UserClaims{
			Id: 123,
		})
		context.Next()
	})
	s.db = startup.InitTestDB()
	s.kafkaClient = startup.InitKafka()
	hdl := startup.InitArticleHandler(article.NewGORMArticleDAO(s.db))
	hdl.RegisterRoutes(s.server)
}

func (s *ArticleGORMHandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `articles`").Error
	assert.NoError(s.T(), err)
	s.db.Exec("TRUNCATE TABLE `published_articles`")
	s.db.Exec("TRUNCATE TABLE `interactives`")
}

func (s *ArticleGORMHandlerTestSuite) TestArticleHandler_Edit() {
	t := s.T()
	testCases := []struct {
		name string
		// 要提前准备数据
		before func(t *testing.T)
		// 验证并且删除数据
		after func(t *testing.T)
		// 构造请求，直接使用 req
		// 也就是说，我们放弃测试 Bind 的异常分支
		req article.Article

		// 预期响应
		wantCode   int
		wantResult Result[int64]
	}{
		{
			name: "新建帖子",
			before: func(t *testing.T) {
				// 什么也不需要做
			},
			after: func(t *testing.T) {
				// 验证一下数据
				var art article.Article
				s.db.Where("author_id = ?", 123).First(&art)
				assert.True(t, art.Ctime > 0)
				assert.True(t, art.Utime > 0)
				// 重置了这些值，因为无法比较
				art.Utime = 0
				art.Ctime = 0
				assert.Equal(t, article.Article{
					Id:       1,
					Title:    "hello，你好",
					Content:  "随便试试",
					AuthorId: 123,
					Status:   domain.ArticleStatusUnpublished.ToUint8(),
				}, art)
			},
			req: article.Article{
				Title:   "hello，你好",
				Content: "随便试试",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Data: 1,
			},
		},
		{
			// 这个是已经有了，然后修改之后再保存
			name: "更新帖子",
			before: func(t *testing.T) {
				// 模拟已经存在的帖子，并且是已经发布的帖子
				s.db.Create(&article.Article{
					Id:       2,
					Title:    "我的标题",
					Content:  "我的内容",
					Ctime:    456,
					Utime:    234,
					AuthorId: 123,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				})
			},
			after: func(t *testing.T) {
				// 验证一下数据
				var art article.Article
				s.db.Where("id = ?", 2).First(&art)
				assert.True(t, art.Utime > 234)
				art.Utime = 0
				assert.Equal(t, article.Article{
					Id:       2,
					Title:    "新的标题",
					Content:  "新的内容",
					AuthorId: 123,
					// 创建时间没变
					Ctime:  456,
					Status: domain.ArticleStatusUnpublished.ToUint8(),
				}, art)
			},
			req: article.Article{
				Id:      2,
				Title:   "新的标题",
				Content: "新的内容",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Data: 2,
			},
		},

		{
			name: "更新别人的帖子",
			before: func(t *testing.T) {
				// 模拟已经存在的帖子
				s.db.Create(&article.Article{
					Id:      3,
					Title:   "我的标题",
					Content: "我的内容",
					Ctime:   456,
					Utime:   234,
					// 注意。这个 AuthorID 设置为另外一个人的ID
					// 意味着在修改别人的帖子
					AuthorId: 789,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				})
			},
			after: func(t *testing.T) {
				// 更新应该是失败了，数据没有发生变化
				var art article.Article
				s.db.Where("id = ?", 3).First(&art)
				assert.Equal(t, article.Article{
					Id:       3,
					Title:    "我的标题",
					Content:  "我的内容",
					Ctime:    456,
					Utime:    234,
					AuthorId: 789,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				}, art)
			},
			req: article.Article{
				Id:      3,
				Title:   "新的标题",
				Content: "新的内容",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Code: 5,
				Msg:  "系统错误",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			data, err := json.Marshal(tc.req)
			// 不能有 error
			assert.NoError(t, err)
			req, err := http.NewRequest(http.MethodPost,
				"/articles/edit", bytes.NewReader(data))
			assert.NoError(t, err)
			req.Header.Set("Content-Type",
				"application/json")
			recorder := httptest.NewRecorder()

			s.server.ServeHTTP(recorder, req)
			code := recorder.Code
			assert.Equal(t, tc.wantCode, code)
			if code != http.StatusOK {
				return
			}
			// 反序列化为结果
			// 利用泛型来限定结果必须是 int64
			var result Result[int64]
			err = json.Unmarshal(recorder.Body.Bytes(), &result)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantResult, result)
			tc.after(t)
		})
	}
}

func (s *ArticleGORMHandlerTestSuite) TestArticle_Publish() {
	t := s.T()

	testCases := []struct {
		name string
		// 要提前准备数据
		before func(t *testing.T)
		// 验证并且删除数据
		after func(t *testing.T)
		req   article.Article

		// 预期响应
		wantCode   int
		wantResult Result[int64]
	}{
		{
			name: "新建帖子并发表",
			before: func(t *testing.T) {
				// 什么也不需要做
			},
			after: func(t *testing.T) {
				// 验证一下数据
				var art article.Article
				s.db.Where("author_id = ?", 123).First(&art)
				assert.Equal(t, "hello，你好", art.Title)
				assert.Equal(t, "随便试试", art.Content)
				assert.Equal(t, int64(123), art.AuthorId)
				assert.True(t, art.Ctime > 0)
				assert.True(t, art.Utime > 0)
				var publishedArt article.PublishedArticle
				s.db.Where("author_id = ?", 123).First(&publishedArt)
				assert.Equal(t, "hello，你好", publishedArt.Title)
				assert.Equal(t, "随便试试", publishedArt.Content)
				assert.Equal(t, int64(123), publishedArt.AuthorId)
				assert.True(t, publishedArt.Ctime > 0)
				assert.True(t, publishedArt.Utime > 0)
			},
			req: article.Article{
				Title:   "hello，你好",
				Content: "随便试试",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Data: 1,
			},
		},
		{
			// 制作库有，但是线上库没有
			name: "更新帖子并新发表",
			before: func(t *testing.T) {
				// 模拟已经存在的帖子
				s.db.Create(&article.Article{
					Id:       2,
					Title:    "我的标题",
					Content:  "我的内容",
					Ctime:    456,
					Utime:    234,
					AuthorId: 123,
				})
			},
			after: func(t *testing.T) {
				// 验证一下数据
				var art article.Article
				s.db.Where("id = ?", 2).First(&art)
				assert.Equal(t, "新的标题", art.Title)
				assert.Equal(t, "新的内容", art.Content)
				assert.Equal(t, int64(123), art.AuthorId)
				// 创建时间没变
				assert.Equal(t, int64(456), art.Ctime)
				// 更新时间变了
				assert.True(t, art.Utime > 234)
				var publishedArt article.PublishedArticle
				s.db.Where("id = ?", 2).First(&publishedArt)
				assert.Equal(t, "新的标题", art.Title)
				assert.Equal(t, "新的内容", art.Content)
				assert.Equal(t, int64(123), art.AuthorId)
				assert.True(t, publishedArt.Ctime > 0)
				assert.True(t, publishedArt.Utime > 0)
			},
			req: article.Article{
				Id:      2,
				Title:   "新的标题",
				Content: "新的内容",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Data: 2,
			},
		},
		{
			name: "更新帖子，并且重新发表",
			before: func(t *testing.T) {
				art := article.Article{
					Id:       3,
					Title:    "我的标题",
					Content:  "我的内容",
					Ctime:    456,
					Utime:    234,
					AuthorId: 123,
				}
				s.db.Create(&art)
				part := article.PublishedArticle{Article: art}
				s.db.Create(&part)
			},
			after: func(t *testing.T) {
				var art article.Article
				s.db.Where("id = ?", 3).First(&art)
				assert.Equal(t, "新的标题", art.Title)
				assert.Equal(t, "新的内容", art.Content)
				assert.Equal(t, int64(123), art.AuthorId)
				// 创建时间没变
				assert.Equal(t, int64(456), art.Ctime)
				// 更新时间变了
				assert.True(t, art.Utime > 234)

				var part article.PublishedArticle
				s.db.Where("id = ?", 3).First(&part)
				assert.Equal(t, "新的标题", part.Title)
				assert.Equal(t, "新的内容", part.Content)
				assert.Equal(t, int64(123), part.AuthorId)
				// 创建时间没变
				assert.Equal(t, int64(456), part.Ctime)
				// 更新时间变了
				assert.True(t, part.Utime > 234)
			},
			req: article.Article{
				Id:      3,
				Title:   "新的标题",
				Content: "新的内容",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Data: 3,
			},
		},
		{
			name: "更新别人的帖子，并且发表失败",
			before: func(t *testing.T) {
				art := article.Article{
					Id:      4,
					Title:   "我的标题",
					Content: "我的内容",
					Ctime:   456,
					Utime:   234,
					// 注意。这个 AuthorID 设置为另外一个人的ID
					AuthorId: 789,
				}
				s.db.Create(&art)
				part := article.PublishedArticle{Article: article.Article{
					Id:       4,
					Title:    "我的标题",
					Content:  "我的内容",
					Ctime:    456,
					Utime:    234,
					AuthorId: 789,
				},
				}
				s.db.Create(&part)
			},
			after: func(t *testing.T) {
				// 更新应该是失败了，数据没有发生变化
				var art article.Article
				s.db.Where("id = ?", 4).First(&art)
				assert.Equal(t, "我的标题", art.Title)
				assert.Equal(t, "我的内容", art.Content)
				assert.Equal(t, int64(456), art.Ctime)
				assert.Equal(t, int64(234), art.Utime)
				assert.Equal(t, int64(789), art.AuthorId)

				var part article.PublishedArticle
				// 数据没有变化
				s.db.Where("id = ?", 4).First(&part)
				assert.Equal(t, "我的标题", part.Title)
				assert.Equal(t, "我的内容", part.Content)
				assert.Equal(t, int64(789), part.AuthorId)
				// 创建时间没变
				assert.Equal(t, int64(456), part.Ctime)
				// 更新时间变了
				assert.Equal(t, int64(234), part.Utime)
			},
			req: article.Article{
				Id:      4,
				Title:   "新的标题",
				Content: "新的内容",
			},
			wantCode: 200,
			wantResult: Result[int64]{
				Code: 5,
				Msg:  "系统错误",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.before(t)
			data, err := json.Marshal(tc.req)
			// 不能有 error
			assert.NoError(t, err)
			req, err := http.NewRequest(http.MethodPost,
				"/articles/publish", bytes.NewReader(data))
			assert.NoError(t, err)
			req.Header.Set("Content-Type",
				"application/json")
			recorder := httptest.NewRecorder()

			s.server.ServeHTTP(recorder, req)
			code := recorder.Code
			assert.Equal(t, tc.wantCode, code)
			if code != http.StatusOK {
				return
			}
			// 反序列化为结果
			// 利用泛型来限定结果必须是 int64
			var result Result[int64]
			err = json.Unmarshal(recorder.Body.Bytes(), &result)
			assert.NoError(t, err)
			assert.Equal(t, tc.wantResult, result)
			tc.after(t)
		})
	}
}

func (s *ArticleGORMHandlerTestSuite) TestPubDetail() {
	testCases := []struct {
		name   string
		before func(t *testing.T)
		after  func(t *testing.T)
		id     int64
		// 预期响应
		wantCode   int
		wantResult Result[web.ArticleVo]
	}{
		{
			name: "查找成功",
			id:   1,
			before: func(t *testing.T) {
				// 准备数据，首先准备文章数据
				err := s.db.Create(&article.PublishedArticle{Article: article.Article{
					Id:       1,
					Title:    "我的标题",
					Content:  "我的内容",
					AuthorId: 123,
					Status:   domain.ArticleStatusPublished.ToUint8(),
					Ctime:    123,
					Utime:    234,
				}}).Error
				assert.NoError(t, err)
				// 准备点赞、收藏的数据
				err = s.db.Create(&dao.Interactive{
					Id:         1,
					BizId:      1,
					Biz:        "article",
					ReadCnt:    1,
					CollectCnt: 2,
					LikeCnt:    3,
				}).Error
				assert.NoError(t, err)
			},
			//after: func(t *testing.T) {
			//	// 需要确保，MQ 里面有这个消息
			//	// 所以需要拿到一条消息
			//	consumer, err := sarama.NewConsumerGroupFromClient("test_group1", s.kafkaClient)
			//	assert.NoError(t, err)
			//	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			//	defer cancel()
			//	err = consumer.Consume(ctx, []string{}, saramax.HandlerFunc(func(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
			//		select {
			//		case <-ctx.Done():
			//			return ctx.Err()
			//		case msg := <-claim.Messages():
			//			// 等一条消息就可以了
			//			// 这里进一步确认 msg 的内容
			//			var evt article2.ReadEvent
			//			err := json.Unmarshal(msg.Value, &evt)
			//			if err != nil {
			//				return err
			//			}
			//			assert.Equal(t, article2.ReadEvent{
			//				Aid: 1,
			//				Uid: 123,
			//			}, evt)
			//		}
			//		return nil
			//	}))
			//	assert.NoError(t, err)
			//},
			wantCode: 200,
			wantResult: Result[web.ArticleVo]{
				Code: 0,
				Msg:  "",
				Data: web.ArticleVo{
					Id:      1,
					Title:   "我的标题",
					Status:  domain.ArticleStatusPublished.ToUint8(),
					Content: "我的内容",
					// 要把作者信息带出去
					Author:     "",
					Ctime:      "",
					Utime:      "",
					ReadCnt:    1,
					CollectCnt: 2,
					LikeCnt:    3,
					Liked:      false,
					Collected:  false,
				},
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			tc.before(t)
			req, err := http.NewRequest(http.MethodGet,
				fmt.Sprintf("/articles/pub/%d", tc.id), nil)
			assert.NoError(t, err)
			req.Header.Set("Content-Type",
				"application/json")
			recorder := httptest.NewRecorder()

			s.server.ServeHTTP(recorder, req)
			code := recorder.Code
			assert.Equal(t, tc.wantCode, code)

			var result Result[web.ArticleVo]
			err = json.Unmarshal(recorder.Body.Bytes(), &result)
			result.Data.Ctime = ""
			result.Data.Utime = ""

			assert.NoError(t, err)
			assert.Equal(t, tc.wantResult, result)
			//tc.after(t)
		})
	}
}

func TestGORMArticle(t *testing.T) {
	suite.Run(t, new(ArticleGORMHandlerTestSuite))
}

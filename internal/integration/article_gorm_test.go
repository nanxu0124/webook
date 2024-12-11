package integration

import (
	"bytes"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"testing"
	"webook/internal/domain"
	"webook/internal/integration/startup"
	"webook/internal/repository/dao/article"
	ijwt "webook/internal/web/jwt"
)

type ArticleGORMHandlerTestSuite struct {
	suite.Suite
	server *gin.Engine
	db     *gorm.DB
}

var author_id int64 = 123

func (s *ArticleGORMHandlerTestSuite) SetupSuite() {
	s.server = gin.Default()
	s.db = startup.InitTestDB()
	s.server.Use(func(context *gin.Context) {
		// 直接设置好
		context.Set("user", ijwt.UserClaims{
			Id: author_id,
		})
		context.Next()
	})
	hdl := startup.InitArticleHandler(article.NewGORMArticleDAO(s.db))
	hdl.RegisterRoutes(s.server)
}

func (s *ArticleGORMHandlerTestSuite) TearDownTest() {
	err := s.db.Exec("TRUNCATE TABLE `articles`").Error
	assert.NoError(s.T(), err)

	err = s.db.Exec("TRUNCATE TABLE `published_articles`").Error
	assert.NoError(s.T(), err)
}

func TestArticle(t *testing.T) {
	suite.Run(t, new(ArticleGORMHandlerTestSuite))
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
		req Article

		// 预期响应
		wantCode   int
		wantResult Result[int64]
	}{
		{
			name: "新建帖子",
			before: func(t *testing.T) {

			},

			after: func(t *testing.T) {
				// 验证数据库
				var art article.Article
				s.db.Where("author_id = ?", 123).First(&art)
				assert.True(t, art.Ctime > 0)
				assert.True(t, art.Utime > 0)
				// 重置了这些值，因为无法比较
				art.Utime = 0
				art.Ctime = 0
				assert.Equal(t, article.Article{
					Id:       1,
					Title:    "测试标题",
					Content:  "测试内容",
					AuthorId: author_id,
					Status:   domain.ArticleStatusUnpublished.ToUint8(),
				}, art)
			},

			req: Article{
				Title:   "测试标题",
				Content: "测试内容",
			},
			wantCode: http.StatusOK,
			wantResult: Result[int64]{
				Data: 1,
			},
		},
		{
			name: "更新帖子",
			before: func(t *testing.T) {
				s.db.Create(&article.Article{
					Id:       2,
					Title:    "我的标题",
					Content:  "我的内容",
					Ctime:    456,
					Utime:    234,
					AuthorId: author_id,
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
			req: Article{
				Id:      2,
				Title:   "新的标题",
				Content: "新的内容",
			},
			wantCode: http.StatusOK,
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
			req: Article{
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

func (s *ArticleGORMHandlerTestSuite) TestArticleHandler_Publish() {
	t := s.T()
	testCases := []struct {
		name string
		// 要提前准备数据
		before func(t *testing.T)
		// 验证并且删除数据
		after func(t *testing.T)
		// 构造请求，直接使用 req
		// 也就是说，我们放弃测试 Bind 的异常分支
		req Article

		// 预期响应
		wantCode   int
		wantResult Result[int64]
	}{
		{
			name: "新建并发表",
			before: func(t *testing.T) {

			},

			after: func(t *testing.T) {
				// 验证一下数据
				var art article.Article
				s.db.Where("author_id = ?", author_id).First(&art)
				assert.True(t, art.Ctime > 0)
				assert.True(t, art.Utime > 0)
				art.Utime = 0
				art.Ctime = 0
				assert.Equal(t, article.Article{
					Id:       1,
					Title:    "新建并发表-标题",
					Content:  "新建并发表-内容",
					AuthorId: author_id,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				}, art)

				var publishedArt article.PublishedArticle
				s.db.Where("author_id = ?", author_id).First(&publishedArt)
				assert.True(t, publishedArt.Ctime > 0)
				assert.True(t, publishedArt.Utime > 0)
				publishedArt.Utime = 0
				publishedArt.Ctime = 0
				assert.Equal(t, article.PublishedArticle{
					Article: article.Article{
						Id:       1,
						Title:    "新建并发表-标题",
						Content:  "新建并发表-内容",
						AuthorId: author_id,
						Status:   domain.ArticleStatusPublished.ToUint8(),
					},
				}, publishedArt)

			},
			req: Article{
				Title:   "新建并发表-标题",
				Content: "新建并发表-内容",
			},
			wantCode: http.StatusOK,
			wantResult: Result[int64]{
				Data: 1,
			},
		},
		{
			// 制作库有，但是线上库没有
			name: "更新帖子并发表",
			before: func(t *testing.T) {
				s.db.Create(&article.Article{
					Id:       2,
					Title:    "更新帖子并发表-标题",
					Content:  "更新帖子并发表-内容",
					Ctime:    456,
					Utime:    234,
					AuthorId: author_id,
					Status:   domain.ArticleStatusUnpublished.ToUint8(),
				})
			},

			after: func(t *testing.T) {
				// 验证一下数据
				var art article.Article
				s.db.Where("id = ?", 2).First(&art)
				assert.Equal(t, int64(456), art.Ctime)
				// 更新时间变了
				assert.True(t, art.Utime > 234)
				art.Utime = 0
				art.Ctime = 0
				assert.Equal(t, article.Article{
					Id:       2,
					Title:    "更新帖子并发表-新标题",
					Content:  "更新帖子并发表-新标题",
					AuthorId: author_id,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				}, art)

				var publishedArt article.PublishedArticle
				s.db.Where("id = ?", 2).First(&publishedArt)
				assert.True(t, publishedArt.Ctime > 0)
				assert.True(t, publishedArt.Utime > 0)
				publishedArt.Utime = 0
				publishedArt.Ctime = 0
				assert.Equal(t, article.PublishedArticle{
					Article: article.Article{
						Id:       2,
						Title:    "更新帖子并发表-新标题",
						Content:  "更新帖子并发表-新标题",
						AuthorId: author_id,
						Status:   domain.ArticleStatusPublished.ToUint8(),
					},
				}, publishedArt)

			},
			req: Article{
				Id:      2,
				Title:   "更新帖子并发表-新标题",
				Content: "更新帖子并发表-新标题",
			},
			wantCode: http.StatusOK,
			wantResult: Result[int64]{
				Data: 2,
			},
		},
		{
			// 制作库和线上库都有
			name: "更新帖子并重新发表",
			before: func(t *testing.T) {
				art := article.Article{
					Id:       2,
					Title:    "更新帖子并重新发表-标题",
					Content:  "更新帖子并重新发表-内容",
					Ctime:    456,
					Utime:    234,
					AuthorId: author_id,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				}
				s.db.Create(&art)
				s.db.Create(&article.PublishedArticle{Article: art})
			},

			after: func(t *testing.T) {
				// 验证一下数据
				var art article.Article
				s.db.Where("id = ?", 2).First(&art)
				assert.Equal(t, int64(456), art.Ctime)
				// 更新时间变了
				assert.True(t, art.Utime > 234)
				art.Utime = 0
				art.Ctime = 0
				assert.Equal(t, article.Article{
					Id:       2,
					Title:    "更新帖子并重新发表-新标题",
					Content:  "更新帖子并重新发表-新标题",
					AuthorId: author_id,
					Status:   domain.ArticleStatusPublished.ToUint8(),
				}, art)

				var publishedArt article.PublishedArticle
				s.db.Where("id = ?", 2).First(&publishedArt)
				assert.True(t, publishedArt.Ctime > 0)
				assert.True(t, publishedArt.Utime > 0)
				publishedArt.Utime = 0
				publishedArt.Ctime = 0
				assert.Equal(t, article.PublishedArticle{
					Article: article.Article{
						Id:       2,
						Title:    "更新帖子并重新发表-新标题",
						Content:  "更新帖子并重新发表-新标题",
						AuthorId: author_id,
						Status:   domain.ArticleStatusPublished.ToUint8(),
					},
				}, publishedArt)

			},
			req: Article{
				Id:      2,
				Title:   "更新帖子并重新发表-新标题",
				Content: "更新帖子并重新发表-新标题",
			},
			wantCode: http.StatusOK,
			wantResult: Result[int64]{
				Data: 2,
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
					Status:   domain.ArticleStatusPublished.ToUint8(),
				}
				s.db.Create(&art)
				s.db.Create(&article.PublishedArticle{Article: art})
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
			req: Article{
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

type Article struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

package dao

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type Article struct {
	Id int64
	// 标题的长度
	// 正常都不会超过这个长度
	Title   string `gorm:"type=varchar(4096)"`
	Content string `gorm:"type=BLOB"`
	// 作者
	AuthorId int64 `gorm:"index"`
	Status   uint8 `gorm:"default=1"`
	Ctime    int64
	Utime    int64
}

type PublishedArticle struct {
	Article
}

var ErrPossibleIncorrectAuthor = errors.New("用户在尝试操作非本人数据")

type ArticleDAO interface {
	Create(ctx context.Context, art Article) (int64, error)
	UpdateById(ctx context.Context, art Article) error
	Sync(ctx context.Context, art Article) (int64, error)
	SyncClosure(ctx context.Context, art Article) (int64, error)
	SyncStatus(ctx context.Context, uid, id int64, status uint8) error
}

type GORMArticleDAO struct {
	db *gorm.DB
}

func NewGORMArticleDAO(db *gorm.DB) ArticleDAO {
	return &GORMArticleDAO{
		db: db,
	}
}

func (dao *GORMArticleDAO) SyncStatus(ctx context.Context, uid, id int64, status uint8) error {
	now := time.Now().UnixMilli()
	return dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&Article{}).
			Where("id= ? AND author_id = ?", id, uid).
			Updates(map[string]interface{}{
				"status": status,
				"utime":  now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrPossibleIncorrectAuthor
		}

		res = tx.Model(&PublishedArticle{}).
			Where("id= ? AND author_id = ?", id, uid).
			Updates(map[string]interface{}{
				"status": status,
				"utime":  now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return ErrPossibleIncorrectAuthor
		}
		return nil
	})
}

// Sync 同步 Article 数据到数据库，并在发布文章表中同步数据
func (dao *GORMArticleDAO) Sync(ctx context.Context, art Article) (int64, error) {
	// 开始一个事务
	tx := dao.db.WithContext(ctx).Begin()

	// 获取当前时间戳，用于文章创建和更新时间
	now := time.Now().UnixMilli()

	// 使用 defer 以保证事务在函数结束时回滚，防止发生错误时数据不一致
	defer tx.Rollback()

	// 使用事务创建一个新的 GORMArticleDAO 实例，确保对数据操作在事务中进行
	txDAO := NewGORMArticleDAO(tx)

	var (
		id  = art.Id // 获取文章的 ID
		err error    // 用于存储操作可能产生的错误
	)

	// 如果文章 ID 为 0，表示是新文章，需要插入数据库；否则，更新已有文章
	if id == 0 {
		// 调用 Create 方法插入新文章，返回插入的文章 ID 和错误信息
		id, err = txDAO.Create(ctx, art)
	} else {
		// 如果文章 ID 不为 0，则执行更新操作
		err = txDAO.UpdateById(ctx, art)
	}

	// 如果插入或更新过程中发生错误，返回错误
	if err != nil {
		return 0, err
	}

	// 更新文章对象的 ID（插入新文章或更新已有文章后，ID 可能已改变）
	art.Id = id

	// 创建一个发布文章对象，用于同步到发布文章表
	publishArt := PublishedArticle{
		Article: art, // 将文章数据复制到发布文章对象中
	}
	// 设置发布时间和更新时间
	publishArt.Utime = now
	publishArt.Ctime = now

	// 使用事务插入或更新发布文章表（通过 OnConflict 处理 ID 冲突）
	err = tx.Clauses(clause.OnConflict{
		// 设置当 ID 冲突时执行更新操作
		Columns: []clause.Column{{Name: "id"}}, // 指定冲突的列（ID）
		DoUpdates: clause.Assignments(map[string]interface{}{
			"title":   art.Title,   // 如果发生冲突，更新文章的标题
			"content": art.Content, // 更新文章的内容
			"status":  art.Status,  // 更新文章的状态
			"utime":   now,         // 更新更新时间
		}),
	}).Create(&publishArt).Error // 在发布文章表中创建或更新数据

	// 如果插入或更新发布文章表时发生错误，返回错误
	if err != nil {
		return 0, err
	}

	// 提交事务，确保所有操作都成功
	tx.Commit()

	// 返回文章的 ID 和可能发生的错误
	return id, tx.Error
}

// SyncClosure 同步文章数据到文章表和发布文章表。
// 文章表存储原始的文章数据，发布文章表存储已发布的文章数据
// 该方法使用事务，确保在一个原子操作中完成所有数据库操作。
func (dao *GORMArticleDAO) SyncClosure(ctx context.Context, art Article) (int64, error) {
	var (
		id = art.Id // 存储文章的 ID
	)

	// 开始一个事务，确保操作的原子性
	err := dao.db.Transaction(func(tx *gorm.DB) error {
		var err error
		now := time.Now().UnixMilli() // 获取当前时间戳

		// 创建一个新的 GORMArticleDAO 实例，使用当前事务的 DB 实例
		txDAO := NewGORMArticleDAO(tx)

		// 判断文章 ID 是否为 0，如果是则表示新文章，执行插入操作
		if id == 0 {
			id, err = txDAO.Create(ctx, art)
		} else {
			// 如果 ID 不为 0，执行更新操作
			err = txDAO.UpdateById(ctx, art)
		}
		// 如果插入或更新文章失败，返回错误
		if err != nil {
			return err
		}

		// 更新文章 ID
		art.Id = id

		// 创建发布文章对象，将原始文章数据传入
		publishArt := PublishedArticle{
			Article: art,
		}
		// 设置发布时间和更新时间
		publishArt.Utime = now
		publishArt.Ctime = now

		// 使用事务操作发布文章表，使用 OnConflict 处理 ID 冲突
		// 如果 ID 已存在，则更新字段 (title, content, utime)
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}}, // 设置冲突的列为 ID
			DoUpdates: clause.Assignments(map[string]interface{}{
				"title":   art.Title,   // 更新标题
				"content": art.Content, // 更新内容
				"utime":   now,         // 更新时间
				"status":  art.Status,
			}),
		}).Create(&publishArt).Error // 如果没有冲突则插入新的发布文章
	})

	// 返回文章 ID 和可能发生的错误
	return id, err
}

func (dao *GORMArticleDAO) Create(ctx context.Context, art Article) (int64, error) {
	now := time.Now().UnixMilli()
	art.Ctime = now
	art.Utime = now
	err := dao.db.WithContext(ctx).Create(&art).Error
	return art.Id, err
}

func (dao *GORMArticleDAO) UpdateById(ctx context.Context, art Article) error {
	now := time.Now().UnixMilli()
	res := dao.db.Model(&Article{}).WithContext(ctx).
		Where("id=? AND author_id = ? ", art.Id, art.AuthorId).
		Updates(map[string]any{
			"title":   art.Title,
			"content": art.Content,
			"status":  art.Status,
			"utime":   now,
		})
	err := res.Error
	if err != nil {
		return err
	}
	if res.RowsAffected == 0 {
		return errors.New("更新数据失败")
	}
	return nil
}

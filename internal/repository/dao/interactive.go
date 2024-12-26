package dao

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type InteractiveDAO interface {
	IncrReadCnt(ctx context.Context, biz string, bizId int64) error
	InsertLikeInfo(ctx context.Context, biz string, bizId, uid int64) error
	GetLikeInfo(ctx context.Context, biz string, bizId, uid int64) (UserLikeBiz, error)
	DeleteLikeInfo(ctx context.Context, biz string, bizId, uid int64) error
	Get(ctx context.Context, biz string, bizId int64) (Interactive, error)
	InsertCollectionBiz(ctx context.Context, cb UserCollectionBiz) error
	GetCollectionInfo(ctx context.Context, biz string, bizId, uid int64) (UserCollectionBiz, error)
}

type GORMInteractiveDAO struct {
	db *gorm.DB
}

func NewGORMInteractiveDAO(db *gorm.DB) InteractiveDAO {
	return &GORMInteractiveDAO{
		db: db,
	}
}

// IncrReadCnt 增加数据库中的阅读计数
func (dao *GORMInteractiveDAO) IncrReadCnt(ctx context.Context, biz string, bizId int64) error {
	// 获取当前时间戳（毫秒）
	now := time.Now().UnixMilli()

	// 使用事务来更新数据库中的阅读计数
	return dao.db.WithContext(ctx).Clauses(clause.OnConflict{
		// 通过 OnConflict 子句确保在记录冲突时更新，而不是插入新记录
		DoUpdates: clause.Assignments(map[string]any{
			"read_cnt": gorm.Expr("`read_cnt`+1"), // 增加阅读计数字段（自增）
			"utime":    now,                       // 更新时间字段
		}),
	}).Create(&Interactive{
		ReadCnt: 1,     // 如果记录不存在，插入时设置初始阅读计数为 1
		Ctime:   now,   // 设置创建时间为当前时间戳
		Utime:   now,   // 设置更新时间为当前时间戳
		Biz:     biz,   // 设置业务标识
		BizId:   bizId, // 设置业务 ID
	}).Error
}

// InsertLikeInfo 插入点赞信息，如果记录已存在，则更新相应的字段
// 参数:
//
//	ctx: 上下文对象，用于控制请求生命周期（例如取消或超时）
//	biz: 业务标识，用于区分不同业务（例如：文章、视频等）
//	bizId: 业务ID，用于标识特定的业务对象（例如：文章ID、视频ID等）
//	uid: 用户ID，用于标识哪个用户进行了点赞操作
func (dao *GORMInteractiveDAO) InsertLikeInfo(ctx context.Context, biz string, bizId, uid int64) error {
	// 获取当前时间戳（毫秒）
	now := time.Now().UnixMilli()

	// 使用事务进行操作，确保这两个插入/更新操作要么都成功，要么都失败
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		// 插入 UserLikeBiz 记录，如果已存在则更新
		err := tx.Clauses(clause.OnConflict{
			// 如果记录冲突（即已经存在），则执行更新操作
			DoUpdates: clause.Assignments(map[string]any{
				"status": 1,   // 更新状态为 1，表示点赞
				"utime":  now, // 更新更新时间为当前时间
			}),
		}).Create(&UserLikeBiz{
			Uid:    uid,   // 用户ID
			Ctime:  now,   // 创建时间
			Utime:  now,   // 更新时间
			Biz:    biz,   // 业务标识（例如：文章、视频）
			BizId:  bizId, // 业务ID（例如：文章ID、视频ID）
			Status: 1,     // 初始状态为 1（表示点赞）
		}).Error
		// 如果插入或更新失败，返回错误
		if err != nil {
			return err
		}

		// 插入 Interactive 记录，如果已存在则更新
		return tx.Clauses(clause.OnConflict{
			// 如果记录冲突（即已经存在），则执行更新操作
			DoUpdates: clause.Assignments(map[string]any{
				"like_cnt": gorm.Expr("`like_cnt`+1"), // 点赞数量加 1
				"utime":    now,                       // 更新时间为当前时间
			}),
		}).Create(&Interactive{
			LikeCnt: 1,     // 初始点赞数量为 1
			Ctime:   now,   // 创建时间
			Utime:   now,   // 更新时间
			Biz:     biz,   // 业务标识
			BizId:   bizId, // 业务ID
		}).Error
	})

	// 返回事务执行的错误（如果有的话）
	return err
}

func (dao *GORMInteractiveDAO) GetLikeInfo(ctx context.Context, biz string, bizId, uid int64) (UserLikeBiz, error) {
	var res UserLikeBiz
	err := dao.db.WithContext(ctx).
		Where("biz=? AND biz_id = ? AND uid = ? AND status = ?", biz, bizId, uid, 1).
		First(&res).Error
	return res, err
}

// DeleteLikeInfo 删除点赞信息，首先将 UserLikeBiz 中的状态设置为 0（取消点赞），
// 然后更新 Interactive 表中的点赞数量。
// 参数:
//
//	ctx: 上下文对象，用于控制请求生命周期（例如取消或超时）
//	biz: 业务标识，用于区分不同业务（例如：文章、视频等）
//	bizId: 业务ID，用于标识特定的业务对象（例如：文章ID、视频ID等）
//	uid: 用户ID，用于标识哪个用户取消了点赞操作
func (dao *GORMInteractiveDAO) DeleteLikeInfo(ctx context.Context, biz string, bizId, uid int64) error {
	// 获取当前时间戳（毫秒），用于更新时间字段
	now := time.Now().UnixMilli()

	// 使用事务进行操作，确保这两个数据库更新操作要么都成功，要么都失败
	err := dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 更新 UserLikeBiz 表中的点赞记录，将 status 设置为 0（表示取消点赞）
		err := tx.Model(&UserLikeBiz{}).
			Where("biz = ? AND biz_id = ? AND uid = ?", biz, bizId, uid).
			Updates(map[string]any{
				"status": 0,   // 状态设置为 0，表示取消点赞
				"utime":  now, // 更新时间为当前时间
			}).Error
		if err != nil {
			// 如果更新失败，返回错误
			return err
		}

		// 更新 Interactive 表中的点赞数量，减少 1
		return dao.db.WithContext(ctx).Clauses(clause.OnConflict{
			// 如果记录冲突（即该业务对象的记录已存在），则执行更新操作
			DoUpdates: clause.Assignments(map[string]any{
				"like_cnt": gorm.Expr("`like_cnt`-1"), // 点赞数量减 1
				"utime":    now,                       // 更新时间为当前时间
			}),
		}).Create(&Interactive{
			LikeCnt: 1,     // 点赞数量初始为 1
			Ctime:   now,   // 创建时间
			Utime:   now,   // 更新时间
			Biz:     biz,   // 业务标识
			BizId:   bizId, // 业务ID
		}).Error
	})

	// 返回事务执行的错误（如果有的话）
	return err
}

func (dao *GORMInteractiveDAO) Get(ctx context.Context, biz string, bizId int64) (Interactive, error) {
	var res Interactive
	err := dao.db.WithContext(ctx).
		Where("biz = ? AND biz_id = ?", biz, bizId).
		First(&res).Error
	return res, err
}

// InsertCollectionBiz 插入用户收藏的业务记录，并更新相应的收藏计数
func (dao *GORMInteractiveDAO) InsertCollectionBiz(ctx context.Context, cb UserCollectionBiz) error {
	// 获取当前时间戳（毫秒）
	now := time.Now().UnixMilli()
	cb.Utime = now
	cb.Ctime = now

	// 使用事务保证数据的一致性
	return dao.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 插入用户收藏的业务记录到数据库
		err := dao.db.WithContext(ctx).Create(&cb).Error
		if err != nil {
			return err // 插入失败则返回错误
		}

		// 更新或插入 Interactive 表中的收藏计数（如果记录已存在则更新）
		return tx.Clauses(clause.OnConflict{
			DoUpdates: clause.Assignments(map[string]any{
				"like_cnt": gorm.Expr("`like_cnt`+1"), // 增加收藏计数
				"utime":    now,                       // 更新时间
			}),
		}).Create(&Interactive{
			CollectCnt: 1,        // 初始收藏计数为 1
			Ctime:      now,      // 创建时间为当前时间
			Utime:      now,      // 更新时间为当前时间
			Biz:        cb.Biz,   // 业务类型
			BizId:      cb.BizId, // 业务对象 ID
		}).Error
	})
}

func (dao *GORMInteractiveDAO) GetCollectionInfo(ctx context.Context, biz string, bizId, uid int64) (UserCollectionBiz, error) {
	var res UserCollectionBiz
	err := dao.db.WithContext(ctx).
		Where("biz=? AND biz_id = ? AND uid = ?", biz, bizId, uid).
		First(&res).Error
	return res, err
}

// 正常来说，一张主表和与它有关联关系的表会共用一个DAO，
// 所以我们就用一个 DAO 来操作

type Interactive struct {
	Id         int64  `gorm:"primaryKey,autoIncrement"`
	BizId      int64  `gorm:"uniqueIndex:biz_type_id"`
	Biz        string `gorm:"type:varchar(128);uniqueIndex:biz_type_id"`
	ReadCnt    int64
	CollectCnt int64
	LikeCnt    int64
	Ctime      int64
	Utime      int64
}

// UserLikeBiz 命名无能，用户点赞的某个东西
type UserLikeBiz struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 三个构成唯一索引
	BizId int64  `gorm:"uniqueIndex:biz_type_id_uid"`
	Biz   string `gorm:"type:varchar(128);uniqueIndex:biz_type_id_uid"`
	Uid   int64  `gorm:"uniqueIndex:biz_type_id_uid"`
	// 依旧是只在 DB 层面生效的状态
	// 1- 有效，0-无效。软删除的用法
	Status uint8
	Ctime  int64
	Utime  int64
}

// Collection 收藏夹
type Collection struct {
	Id   int64  `gorm:"primaryKey,autoIncrement"`
	Name string `gorm:"type=varchar(1024)"`
	Uid  int64  `gorm:""`

	Ctime int64
	Utime int64
}

// UserCollectionBiz 收藏的东西
type UserCollectionBiz struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	// 收藏夹 ID
	// 作为关联关系中的外键，我们这里需要索引
	Cid   int64  `gorm:"index"`
	BizId int64  `gorm:"uniqueIndex:biz_type_id_uid"`
	Biz   string `gorm:"type:varchar(128);uniqueIndex:biz_type_id_uid"`
	// 这算是一个冗余，因为正常来说，
	// 只需要在 Collection 中维持住 Uid 就可以
	Uid   int64 `gorm:"uniqueIndex:biz_type_id_uid"`
	Ctime int64
	Utime int64
}

package article

import "context"

type ArticleDAO interface {
	Create(ctx context.Context, art Article) (int64, error)
	UpdateById(ctx context.Context, art Article) error
	Sync(ctx context.Context, art Article) (int64, error)
	SyncClosure(ctx context.Context, art Article) (int64, error)
	SyncStatus(ctx context.Context, uid, id int64, status uint8) error
}

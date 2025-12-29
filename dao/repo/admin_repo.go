package repo

import (
	"context"
	"errors"

	"github.com/zjutjh/mygo/ndb"
	"gorm.io/gorm"

	"app/dao/model"
	"app/dao/query"
)

type AdminRepo struct {
	query *query.Query
}

func NewAdminRepo() *AdminRepo {
	return &AdminRepo{
		query: query.Use(ndb.Pick()),
	}
}

func (r *AdminRepo) FindByUsername(ctx context.Context, username string) (*model.Admin, error) {
	a := r.query.Admin
	record, err := a.WithContext(ctx).Where(a.Username.Eq(username)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return record, err
}

func (r *AdminRepo) FindListByIDs(ctx context.Context, ids []int64) ([]*model.Admin, error) {
	a := r.query.Admin
	return a.WithContext(ctx).Where(a.ID.In(ids...)).Find()
}

func (r *AdminRepo) Create(ctx context.Context, record *model.Admin) error {
	a := r.query.Admin
	return a.WithContext(ctx).Create(record)
}

package repo

import (
	"context"

	"github.com/zjutjh/mygo/ndb"

	"app/comm"
	"app/dao/model"
	"app/dao/query"
)

type SurveyRepo struct {
	query *query.Query
}

func NewSurveyRepo(tx ...*query.Query) *SurveyRepo {
	var q *query.Query
	if len(tx) > 0 {
		q = tx[0]
	} else {
		q = query.Use(ndb.Pick())
	}
	return &SurveyRepo{
		query: q,
	}
}

func (r *SurveyRepo) FindByID(ctx context.Context, id int64) (*model.Survey, error) {
	s := r.query.Survey
	record, err := s.WithContext(ctx).Where(s.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (r *SurveyRepo) FindByPath(ctx context.Context, path string) (*model.Survey, error) {
	s := r.query.Survey
	record, err := s.WithContext(ctx).Where(s.Path.Eq(path)).First()
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (r *SurveyRepo) FindPage(ctx context.Context, page, pageSize int, adminID int64, surveyType comm.SurveyType, status comm.SurveyStatus, keyword string) ([]*model.Survey, int64, error) {
	s := r.query.Survey
	do := s.WithContext(ctx)
	if adminID > 0 {
		do = do.Where(s.AdminID.Eq(adminID))
	}
	if surveyType > 0 {
		do = do.Where(s.Type.Eq(int8(surveyType)))
	}
	if status > 0 {
		do = do.Where(s.Status.Eq(int8(status)))
	}
	if keyword != "" {
		do = do.Where(s.Title.Like("%" + keyword + "%"))
	}

	list, err := do.Omit(s.Schema).Order(s.ID.Desc()).Limit(pageSize).Offset((page - 1) * pageSize).Find()
	if err != nil {
		return nil, 0, err
	}

	total, err := do.Count()
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *SurveyRepo) Create(ctx context.Context, survey *model.Survey) error {
	s := r.query.Survey
	return s.WithContext(ctx).Create(survey)
}

func (r *SurveyRepo) UpdateSchema(ctx context.Context, id int64, title, schema string) (int64, error) {
	s := r.query.Survey
	result, err := s.WithContext(ctx).Where(s.ID.Eq(id)).UpdateSimple(s.Title.Value(title), s.Schema.Value(schema))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected, nil
}

func (r *SurveyRepo) UpdateStatus(ctx context.Context, id int64, status comm.SurveyStatus) (int64, error) {
	s := r.query.Survey
	result, err := s.WithContext(ctx).Where(s.ID.Eq(id)).UpdateSimple(s.Status.Value(int8(status)))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected, nil
}

func (r *SurveyRepo) DeleteByID(ctx context.Context, id int64) (int64, error) {
	s := r.query.Survey
	result, err := s.WithContext(ctx).Where(s.ID.Eq(id)).Delete()
	if err != nil {
		return 0, err
	}
	return result.RowsAffected, nil
}

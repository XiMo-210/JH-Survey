package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"github.com/zjutjh/mygo/nedis"

	"app/dao/model"
)

const (
	SurveyCachePrefix = "survey:"
	SurveyCacheTTL    = 5 * time.Minute
)

type SurveyCache struct {
	rdb redis.UniversalClient
}

func NewSurveyCache() *SurveyCache {
	return &SurveyCache{
		rdb: nedis.Pick(),
	}
}

func (c *SurveyCache) Set(ctx context.Context, path string, survey *model.Survey) error {
	val, err := sonic.MarshalString(survey)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, c.getKey(path), val, SurveyCacheTTL).Err()
}

func (c *SurveyCache) Get(ctx context.Context, path string) (*model.Survey, error) {
	val, err := c.rdb.Get(ctx, c.getKey(path)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var survey model.Survey
	if err := sonic.UnmarshalString(val, &survey); err != nil {
		return nil, err
	}
	return &survey, nil
}

func (c *SurveyCache) Del(ctx context.Context, path string) error {
	return c.rdb.Del(ctx, c.getKey(path)).Err()
}

func (c *SurveyCache) getKey(path string) string {
	return fmt.Sprintf("%s%s", SurveyCachePrefix, path)
}

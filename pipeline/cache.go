package pipeline

import (
	"context"
	"database/sql"

	"github.com/ignisVeneficus/logging"
	"github.com/ignisVeneficus/lumenta/db/dao"
	"github.com/ignisVeneficus/lumenta/db/dbo"
)

func CreateTagCache() TagCache {
	return TagCache{
		m: make(map[string]dbo.TagID),
	}
}

func LoadTagCache(cache *TagCache, database *sql.DB, ctx context.Context) error {
	logScope, ctx := logging.Enter(ctx, "pipeline/cache/tag", nil, nil)
	tags, err := dao.QueryTags(database, ctx)
	if err != nil {
		logging.ExitErr(logScope, err)
		return err
	}
	tagMap := make(map[dbo.TagID]*dbo.Tag, len(tags))
	pathMap := make(map[dbo.TagID]string, len(tags))
	stack := make([]dbo.TagID, len(tags))
	for i, tag := range tags {
		tagMap[*tag.ID] = &tags[i]
		stack[i] = *tag.ID
	}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		if _, ok := pathMap[id]; ok {
			stack = stack[:len(stack)-1]
			continue
		}
		tag := tagMap[id]
		if tag.ParentID == nil {
			path := tag.Name
			pathMap[id] = path
			cache.m[path] = id
			stack = stack[:len(stack)-1]
			continue
		}
		if parentPath, ok := pathMap[*tag.ParentID]; ok {
			path := parentPath + "/" + tag.Name
			pathMap[id] = path
			cache.m[path] = id
			stack = stack[:len(stack)-1]
			continue
		}
		stack = append(stack, *tag.ParentID)
	}
	logging.Exit(logScope, "ok", nil)
	return nil
}

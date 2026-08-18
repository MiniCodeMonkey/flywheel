package mowl

import (
	"context"
	"testing"
)

func TestSegmentCategories(t *testing.T) {
	c := serveFixtures(t, map[string]string{
		"/v1/segmentcategories": "../../testdata/segmentcategories.json",
	})
	cats, err := c.SegmentCategories(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var warmup bool
	for _, cat := range cats {
		if cat.SegmentCategoryID == 11 && cat.IsWarmup {
			warmup = true
		}
	}
	if !warmup {
		t.Fatal("expected Warm up (11) with IsWarmup")
	}
}

func TestSegmentTypeAliasResolves(t *testing.T) {
	if SegmentTypeAlias["cooldown"] != 15 || PositionAlias["seated"] != 1 {
		t.Fatalf("alias maps wrong: %v %v", SegmentTypeAlias, PositionAlias)
	}
}

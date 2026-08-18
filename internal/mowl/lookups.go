package mowl

import "context"

type SegmentCategory struct {
	SegmentCategoryID int    `json:"SegmentCategoryID"`
	Name              string `json:"Name"`
	IsWarmup          bool   `json:"IsWarmup"`
	IsActiveRecovery  bool   `json:"IsActiveRecovery"`
	IsCooldown        bool   `json:"IsCooldown"`
}

type ActivityType struct {
	ActivityTypeID   int    `json:"ActivityTypeID"`
	ActivityTypeName string `json:"ActivityTypeName"`
}

// Friendly spec `type:` → MOWL SegmentCategoryID. Confirm ids vs testdata.
var SegmentTypeAlias = map[string]int{
	"warmup": 11, "intervals": 10, "climb": 12, "tabata": 41,
	"recovery": 14, "cooldown": 15,
}

// Friendly position → PositionTypeID (confirmed 1=Seated; 2=Standing from Task 2).
var PositionAlias = map[string]int{"seated": 1, "standing": 2}

func (c *Client) SegmentCategories(ctx context.Context) ([]SegmentCategory, error) {
	var out []SegmentCategory
	err := c.do(ctx, "GET", "/v1/segmentcategories", nil, &out)
	return out, err
}

func (c *Client) ActivityTypes(ctx context.Context) ([]ActivityType, error) {
	var out []ActivityType
	err := c.do(ctx, "GET", "/v1/ActivityTypes", nil, &out)
	return out, err
}

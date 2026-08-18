package mowl

import (
	"context"
	"fmt"
)

func (c *Client) CreateCategory(ctx context.Context, name string) (int, error) {
	var out struct {
		ProgramCategoryID int `json:"ProgramCategoryID"`
	}
	err := c.do(ctx, "PUT", "/v1/programcategories", map[string]any{
		"Name": name, "IsPublic": false, "IsOfficial": false,
		"HideName": false, "ActivityTypeID": 0,
	}, &out)
	return out.ProgramCategoryID, err
}

func (c *Client) CreateProgram(ctx context.Context, p Program) (Program, error) {
	var out Program
	err := c.do(ctx, "PUT", "/v1/programs", p, &out)
	return out, err
}

func (c *Client) CreateSegment(ctx context.Context, name string, programID, segmentCategoryID int, f SegmentFlags) (int, error) {
	body := map[string]any{
		"Name": name, "ProgramID": programID,
		"ActivityTypeID": 0, "BikeTypeID": 1, "IsPublic": false,
		"IsWarmup": f.IsWarmup, "IsActiveRecovery": f.IsActiveRecovery, "IsCooldown": f.IsCooldown,
	}
	if segmentCategoryID > 0 {
		body["SegmentCategoryID"] = segmentCategoryID
	}
	var out struct {
		SegmentID int `json:"SegmentID"`
	}
	err := c.do(ctx, "PUT", "/v1/segments", body, &out)
	return out.SegmentID, err
}

func (c *Client) SetIntervals(ctx context.Context, segmentID int, ivs []Interval) error {
	return c.do(ctx, "PUT", fmt.Sprintf("/v1/segments/%d/intervals/multiple", segmentID),
		map[string]any{"ReplaceExisting": true, "Intervals": ivs}, nil)
}

func (c *Client) AttachSegments(ctx context.Context, programID int, segmentIDs []int) error {
	return c.do(ctx, "PUT", fmt.Sprintf("/v1/programs/%d/segments", programID),
		map[string]any{"SegmentIDs": segmentIDs}, nil)
}

// LinkPlaylist binds an imported playlist to a program. Implement per the Task 2
// finding: if PlaylistID on PUT /v1/programs persists the link, re-PUT the
// program with PlaylistID set; otherwise call the dedicated endpoint recorded
// in the findings note.
func (c *Client) LinkPlaylist(ctx context.Context, programID, playlistID int) error {
	var out Program
	return c.do(ctx, "PUT", "/v1/programs",
		Program{ProgramID: programID, PlaylistID: playlistID}, &out)
}

func (c *Client) ProgramTSS(ctx context.Context, programID int) (float64, error) {
	var tss float64
	err := c.do(ctx, "GET", fmt.Sprintf("/v1/calculations/program/%d/TSS", programID), nil, &tss)
	return tss, err
}

func (c *Client) DeleteProgram(ctx context.Context, programID int) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/v1/programs/%d", programID), nil, nil)
}

func (c *Client) DeleteCategory(ctx context.Context, categoryID int) error {
	return c.do(ctx, "DELETE", fmt.Sprintf("/v1/programcategories/%d", categoryID), nil, nil)
}

func (c *Client) MyPrograms(ctx context.Context, creatorID int) ([]Program, error) {
	var out []Program
	err := c.do(ctx, "POST", "/v1/programs",
		map[string]any{"CreatorID": creatorID, "ItemsPerPage": 100, "PageNumber": 1}, &out)
	return out, err
}

// Me returns the current user's UserID from GET /v1/Users/Me.
func (c *Client) Me(ctx context.Context) (int, error) {
	var out struct {
		UserID int `json:"UserID"`
	}
	err := c.do(ctx, "GET", "/v1/Users/Me", nil, &out)
	return out.UserID, err
}

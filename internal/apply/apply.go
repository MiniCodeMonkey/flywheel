package apply

import (
	"context"
	"fmt"

	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/minicodemonkey/flywheel/internal/spec"
)

type MowlAPI interface {
	CreateCategory(ctx context.Context, name string) (int, error)
	MyPrograms(ctx context.Context, creatorID int) ([]mowl.Program, error)
	CreateProgram(ctx context.Context, p mowl.Program) (mowl.Program, error)
	CreateSegment(ctx context.Context, name string, programID, segCatID int, f mowl.SegmentFlags) (int, error)
	SetIntervals(ctx context.Context, segmentID int, ivs []mowl.Interval) error
	AttachSegments(ctx context.Context, programID int, segmentIDs []int) error
	ProgramTSS(ctx context.Context, programID int) (float64, error)
	DeleteProgram(ctx context.Context, programID int) error
}

type Result struct {
	ProgramID  int
	PlaylistID int
	ServerTSS  float64
}

func flagsFor(t string) mowl.SegmentFlags {
	switch t {
	case "warmup":
		return mowl.SegmentFlags{IsWarmup: true}
	case "recovery":
		return mowl.SegmentFlags{IsActiveRecovery: true}
	case "cooldown":
		return mowl.SegmentFlags{IsCooldown: true}
	}
	return mowl.SegmentFlags{}
}

// activityTypeID resolves a course's activity string to the MOWL
// ActivityTypeID. Unknown/empty activity defaults to cycling (0).
func activityTypeID(activity string) int {
	switch activity {
	case "running":
		return 1
	case "cycling":
		return 0
	default:
		return 0
	}
}

func toIntervals(s spec.Segment) ([]mowl.Interval, error) {
	out := make([]mowl.Interval, 0, len(s.Intervals))
	for _, iv := range s.Intervals {
		pos, ok := mowl.PositionAlias[iv.Position]
		if !ok {
			return nil, fmt.Errorf("segment %q: unknown position %q", s.Name, iv.Position)
		}
		out = append(out, mowl.Interval{
			Duration: iv.Duration, RPMFrom: iv.Cadence[0], RPMTo: iv.Cadence[1],
			FTPFrom: iv.Intensity.From, FTPTo: iv.Intensity.To,
			Intensity: (iv.Intensity.From + iv.Intensity.To) / 2, PositionTypeID: pos,
		})
	}
	return out, nil
}

func Apply(ctx context.Context, api MowlAPI, c spec.Course, pl mowl.Playlist, styles spec.Styles, creatorID int) (Result, error) {
	catID, err := api.CreateCategory(ctx, c.Category)
	if err != nil {
		return Result{}, fmt.Errorf("category: %w", err)
	}
	// idempotency: delete an existing same-named program first
	existing, err := api.MyPrograms(ctx, creatorID)
	if err != nil {
		return Result{}, fmt.Errorf("list programs: %w", err)
	}
	for _, p := range existing {
		if p.Name == c.Name {
			if err := api.DeleteProgram(ctx, p.ProgramID); err != nil {
				return Result{}, fmt.Errorf("replace delete: %w", err)
			}
		}
	}
	prog, err := api.CreateProgram(ctx, mowl.Program{
		Name: c.Name, ProgramCategoryID: catID, IsPublic: false,
		ActivityTypeID: activityTypeID(c.Activity), BikeTypeID: 1, Description: styles.Describe(c.Style),
		PlaylistID: pl.PlaylistID,
	})
	if err != nil {
		return Result{}, fmt.Errorf("program: %w", err)
	}
	var segIDs []int
	for _, seg := range c.Segments {
		segCat := mowl.SegmentTypeAlias[seg.Type]
		sid, err := api.CreateSegment(ctx, seg.Name, prog.ProgramID, segCat, flagsFor(seg.Type))
		if err != nil {
			return Result{}, fmt.Errorf("segment %q: %w", seg.Name, err)
		}
		ivs, err := toIntervals(seg)
		if err != nil {
			return Result{}, err
		}
		if err := api.SetIntervals(ctx, sid, ivs); err != nil {
			return Result{}, fmt.Errorf("intervals %q: %w", seg.Name, err)
		}
		segIDs = append(segIDs, sid)
	}
	if err := api.AttachSegments(ctx, prog.ProgramID, segIDs); err != nil {
		return Result{}, fmt.Errorf("attach: %w", err)
	}
	tss, err := api.ProgramTSS(ctx, prog.ProgramID)
	if err != nil {
		return Result{}, fmt.Errorf("tss: %w", err)
	}
	return Result{ProgramID: prog.ProgramID, PlaylistID: pl.PlaylistID, ServerTSS: tss}, nil
}

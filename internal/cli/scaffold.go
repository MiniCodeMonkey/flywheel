package cli

import (
	"fmt"
	"os"

	"github.com/minicodemonkey/flywheel/internal/plan"
	"github.com/minicodemonkey/flywheel/internal/scaffold"
	"github.com/minicodemonkey/flywheel/internal/spec"
	"github.com/spf13/cobra"
)

func newScaffoldCmd() *cobra.Command {
	var (
		segFlags   []string
		name       string
		category   string
		outPath    string
		targetTSS  int
		minSection int
		maxSection int
		endHot     bool
		crossfade  int
		maxStand   int
		standCad   int
		accShare   float64
	)
	cmd := &cobra.Command{
		Use:   "scaffold <spotify-playlist-id>",
		Short: "Generate a course.yaml whose intervals follow the music",
		Long: `Generate a course.yaml from a Spotify playlist.

Each musical section of each track becomes one interval, and a section's
intensity comes from how loud it is relative to the rest of its own track --
so choruses and drops carry the work and verses and bridges back off. The
result is a starting point: edit it, preview it, then apply it.

Segments are given as Name:type:tracks, repeatable and in ride order:

  flywheel scaffold 7ksLrb... \
    --segment "Roll Out:warmup:1-3" \
    --segment "Climbs:climb:4-7" \
    --segment "Recovery:recovery:8" \
    --segment "Finish:intervals:9-10" --tss 75`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(segFlags) == 0 {
				return fmt.Errorf("at least one --segment is required")
			}
			var segs []scaffold.SegmentSpec
			for _, f := range segFlags {
				s, err := scaffold.ParseSegment(f)
				if err != nil {
					return err
				}
				segs = append(segs, s)
			}
			ctx := cmd.Context()
			cl, _, err := newClient(ctx)
			if err != nil {
				return err
			}
			pl, err := cl.SpotifyPlaylist(ctx, args[0])
			if err != nil {
				return err
			}
			var tracks []scaffold.Track
			for i, tr := range pl.Tracks {
				t := scaffold.Track{Index: i + 1, Title: tr.Title, BPM: tr.BPM, DurationSec: tr.DurationMs / 1000}
				secs, err := cl.AudioAnalysis(ctx, tr.SpotifyTrackID)
				if err == nil {
					for _, s := range secs {
						t.Sections = append(t.Sections, scaffold.Section{Duration: s.Duration, Loudness: s.Loudness})
					}
				}
				tracks = append(tracks, t)
			}
			opts := scaffold.Options{TargetTSS: targetTSS, MinSection: minSection,
				MaxSection: maxSection, EndHot: endHot, Crossfade: crossfade,
				MaxStanding: maxStand, StandingCadenceMin: scaffold.Defaults().StandingCadenceMin,
				StandingCadenceMax: standCad, ACCShare: accShare}
			course, gamma, err := scaffold.Build(tracks, segs, opts)
			if err != nil {
				return err
			}
			course.Name, course.Category = name, category
			course.Playlist.CrossfadeSec = &crossfade
			total := 0
			for _, s := range course.Segments {
				for _, iv := range s.Intervals {
					total += iv.Duration
				}
			}
			out := scaffold.Render(course, args[0], (total+30)/60, targetTSS)
			if outPath == "-" {
				fmt.Fprint(cmd.OutOrStdout(), out)
			} else if err := os.WriteFile(outPath, []byte(out), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.ErrOrStderr(),
				"scaffolded %s: %d:%02d, %d segments, estimated TSS %.1f (loudness curve %.2f)\n",
				outPath, total/60, total%60, len(course.Segments), plan.EstimateTSS(course), gamma)
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&segFlags, "segment", nil, `segment as "Name:type:tracks", repeatable, in ride order`)
	cmd.Flags().StringVar(&name, "name", "Scaffolded Ride", "course name")
	cmd.Flags().StringVar(&category, "category", "My Rides", "MOWL category")
	cmd.Flags().StringVarP(&outPath, "out", "o", "course.yaml", `output path, or "-" for stdout`)
	cmd.Flags().IntVar(&targetTSS, "tss", scaffold.Defaults().TargetTSS, "target TSS to tune toward")
	cmd.Flags().IntVar(&minSection, "min-section", scaffold.Defaults().MinSection, "fold sections shorter than this (seconds) into their neighbour")
	cmd.Flags().IntVar(&maxSection, "max-section", scaffold.Defaults().MaxSection, "split sections longer than this (seconds); 0 to never split")
	cmd.Flags().BoolVar(&endHot, "end-hot", true, "end each work segment on its hardest zone")
	cmd.Flags().IntVar(&maxStand, "max-standing", scaffold.Defaults().MaxStanding,
		"longest unbroken standing run (seconds)")
	cmd.Flags().IntVar(&standCad, "standing-cadence-max", scaffold.Defaults().StandingCadenceMax,
		"cadence ceiling while out of the saddle")
	cmd.Flags().Float64Var(&accShare, "acc", scaffold.Defaults().ACCShare,
		"roughly what fraction of work intervals become ACC bursts")
	cmd.Flags().IntVar(&crossfade, "crossfade", spec.DefaultCrossfadeSec,
		"seconds each track overlaps the next; MOWL requires Spotify crossfade at 10")
	return cmd
}

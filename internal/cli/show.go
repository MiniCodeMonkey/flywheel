package cli

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/minicodemonkey/flywheel/internal/mowl"
	"github.com/spf13/cobra"
)

// zoneBandName labels a Coggan zone the way the MOWL app colours it, so a
// read-back program can be compared against the course that produced it.
func zoneBandName(zone int) string {
	switch zone {
	case 1:
		return "white"
	case 2:
		return "blue"
	case 3:
		return "green"
	case 4:
		return "yellow"
	case 5:
		return "red"
	case 6, 7:
		return "fire"
	}
	return "?"
}

// hhmmss renders a duration in seconds as m:ss.
func hhmmss(sec int) string { return fmt.Sprintf("%d:%02d", sec/60, sec%60) }

func segmentRole(s mowl.Segment) string {
	switch {
	case s.IsWarmup:
		return "warmup"
	case s.IsActiveRecovery:
		return "recovery"
	case s.IsCooldown:
		return "cooldown"
	}
	return "work"
}

func newShowCmd() *cobra.Command {
	var intervals bool
	cmd := &cobra.Command{
		Use:   "show <program-id>",
		Short: "Read back a program from MOWL with its segments and intervals",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			programID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("program id: %w", err)
			}
			ctx := cmd.Context()
			cl, _, err := newClient(ctx)
			if err != nil {
				return err
			}
			p, err := cl.Program(ctx, programID)
			if err != nil {
				return err
			}
			tss, err := cl.ProgramTSS(ctx, programID)
			if err != nil {
				return err
			}
			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				b, err := json.MarshalIndent(struct {
					Program mowl.Program `json:"program"`
					TSS     float64      `json:"server_tss"`
				}{p, tss}, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%d  %s\n", p.ProgramID, p.Name)
			if p.CategoryName != "" {
				fmt.Fprintf(out, "category: %s\n", p.CategoryName)
			}
			fmt.Fprintf(out, "%s total, %d segments, server TSS %.1f\n\n",
				hhmmss(p.TotalDuration), len(p.Segments), tss)
			for _, s := range p.Segments {
				last := "-"
				if n := len(s.Intervals); n > 0 {
					last = zoneBandName(s.Intervals[n-1].ScaleCoggan)
				}
				fmt.Fprintf(out, "  %-20s %-8s %7s  %3d intervals  ends %s\n",
					s.Name, segmentRole(s), hhmmss(s.TotalDuration), len(s.Intervals), last)
				if !intervals {
					continue
				}
				at := 0
				for _, iv := range s.Intervals {
					fmt.Fprintf(out, "      %6s  %4ds  %3d-%-3d%%FTP  %3d-%-3d rpm  %-8s %s\n",
						hhmmss(at), iv.Duration, iv.FTPFrom, iv.FTPTo,
						iv.RPMFrom, iv.RPMTo, iv.PositionTypeName, zoneBandName(iv.ScaleCoggan))
					at += iv.Duration
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&intervals, "intervals", false, "also print every interval")
	return cmd
}

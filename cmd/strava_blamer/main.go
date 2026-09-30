package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/droneru/strava_blamer/internal/strava"
)

var errNotImplemented = errors.New("not implemented")

// clientFactory builds a Strava client; tests substitute a fake.
type clientFactory func(ctx context.Context, tokenFile string) (strava.Client, error)

func main() {
	if err := newRootCmd(newStravaClient).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func newStravaClient(ctx context.Context, tokenFile string) (strava.Client, error) {
	cfg := strava.AuthConfig{
		ClientID:     os.Getenv("STRAVA_CLIENT_ID"),
		ClientSecret: os.Getenv("STRAVA_CLIENT_SECRET"),
	}
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, errors.New("STRAVA_CLIENT_ID and STRAVA_CLIENT_SECRET must be set")
	}
	ts, err := strava.NewTokenSource(ctx, cfg, tokenFile, os.Getenv("STRAVA_REFRESH_TOKEN"))
	if err != nil {
		return nil, err
	}
	return strava.NewHTTPClient(oauth2.NewClient(ctx, ts), strava.DefaultBaseURL), nil
}

func newRootCmd(newClient clientFactory) *cobra.Command {
	root := &cobra.Command{
		Use:           "strava_blamer",
		Short:         "Переименовывает активности Strava с названием по умолчанию",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().String("token-file", "token.json", "файл с refresh token Strava")
	root.AddCommand(newGetCmd(newClient), newBlameCmd(), newSweepCmd(), newWatchCmd())
	return root
}

func newGetCmd(newClient clientFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id|URL>",
		Short: "Показать информацию об активности",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strava.ParseActivityRef(args[0])
			if err != nil {
				return err
			}
			tokenFile, _ := cmd.Flags().GetString("token-file")
			client, err := newClient(cmd.Context(), tokenFile)
			if err != nil {
				return err
			}

			a, err := client.GetActivity(cmd.Context(), id)
			if err != nil {
				return err
			}
			var laps []strava.Lap
			if a.SportType == "Swim" {
				if laps, err = client.GetLaps(cmd.Context(), id); err != nil {
					return err
				}
			}
			printActivity(cmd.OutOrStdout(), a, laps)
			return nil
		},
	}
}

func printActivity(w io.Writer, a strava.Activity, laps []strava.Lap) {
	fmt.Fprintf(w, "ID:        %d\n", a.ID)
	fmt.Fprintf(w, "Название:  %s\n", a.Name)
	fmt.Fprintf(w, "Тип:       %s\n", a.SportType)
	fmt.Fprintf(w, "Дистанция: %.0f m\n", a.DistanceM)
	fmt.Fprintf(w, "Начало:    %s\n", a.StartDate.Format(time.RFC3339))
	if a.SportType != "Swim" {
		return
	}
	if len(laps) == 0 {
		fmt.Fprintln(w, "Laps:      нет")
		return
	}
	fmt.Fprintln(w, "Laps:")
	for _, l := range laps {
		fmt.Fprintf(w, "  %3d  %6.0f m  %s\n", l.Index, l.DistanceM, l.MovingTime)
	}
}

func newBlameCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "blame <id|URL>",
		Short: "Обработать одну активность",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
	addDryRunFlag(cmd)
	return cmd
}

func newSweepCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sweep",
		Short: "Обработать необработанные активности за последние N дней",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
	addDryRunFlag(cmd)
	addDaysFlag(cmd)
	return cmd
}

func newWatchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Запускать sweep периодически",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errNotImplemented
		},
	}
	addDryRunFlag(cmd)
	addDaysFlag(cmd)
	cmd.Flags().Duration("interval", time.Hour, "интервал между обходами")
	return cmd
}

func addDryRunFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("dry-run", false, "показать решение без изменений в Strava")
}

func addDaysFlag(cmd *cobra.Command) {
	cmd.Flags().Int("days", 7, "за сколько последних дней обрабатывать активности")
}

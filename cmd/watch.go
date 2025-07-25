package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/agoodway/workie/config"
	"github.com/agoodway/workie/manager"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	watchInterval     string
	watchPort         int
	watchNotifyMethod string
	watchQuiet        bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor worktree branches for potential rebase conflicts",
	Long: `Start a background server that monitors all worktree branches for potential 
conflicts when rebasing on the main branch. Sends notifications when conflicts are detected.`,
	Example: `  # Start the watch server with default settings
  workie watch
  
  # Check every 10 minutes instead of default 5
  workie watch --interval 10m
  
  # Use a custom port
  workie watch --port 8081
  
  # Run in quiet mode
  workie watch --quiet`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse interval duration
		interval, err := time.ParseDuration(watchInterval)
		if err != nil {
			return fmt.Errorf("invalid interval format: %w", err)
		}

		// Validate interval
		if interval < time.Minute {
			return fmt.Errorf("interval must be at least 1 minute")
		}

		// Get the repository root
		repoRoot, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Create manager with options
		opts := manager.Options{
			Quiet:            watchQuiet,
			ShowInitMessages: !watchQuiet,
		}
		wm := manager.NewWithOptions(opts)

		// Detect git repository
		if err := wm.DetectGitRepository(); err != nil {
			return fmt.Errorf("failed to detect git repository: %w", err)
		}

		// Load configuration
		cfg, err := config.LoadConfig(repoRoot, "")
		if err != nil {
			// Config is optional for watch, so just log if verbose
			if !watchQuiet {
				fmt.Printf("⚠️  No configuration file found, using defaults\n")
			}
		} else {
			wm.Config = cfg

			// Override with config values if not specified via flags
			if cmd.Flags().Lookup("interval").Changed == false && cfg.Watch != nil && cfg.Watch.IntervalMinutes > 0 {
				interval = time.Duration(cfg.Watch.IntervalMinutes) * time.Minute
			}
			if cmd.Flags().Lookup("port").Changed == false && cfg.Watch != nil && cfg.Watch.Port > 0 {
				watchPort = cfg.Watch.Port
			}
		}

		// Create watch server
		server := manager.NewWatchServer(wm, manager.WatchServerOptions{
			Port:         watchPort,
			Interval:     interval,
			NotifyMethod: watchNotifyMethod,
			Quiet:        watchQuiet,
		})

		// Set up graceful shutdown
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		go func() {
			<-sigChan
			if !watchQuiet {
				fmt.Println("\n📊 Shutting down watch server...")
			}
			cancel()
		}()

		// Start the server
		if !watchQuiet {
			fmt.Printf("%s Starting workie watch server...\n", color.GreenString("✓"))
			fmt.Printf("📊 Monitoring worktrees every %s\n", interval)
			fmt.Printf("🌐 Server running on http://localhost:%d\n", watchPort)
			fmt.Printf("Press Ctrl+C to stop\n\n")
		}

		if err := server.Start(ctx); err != nil {
			return fmt.Errorf("watch server error: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(watchCmd)

	// Add flags
	watchCmd.Flags().StringVarP(&watchInterval, "interval", "i", "5m", "Check interval (e.g., 5m, 10m, 1h)")
	watchCmd.Flags().IntVarP(&watchPort, "port", "p", 8080, "Server port")
	watchCmd.Flags().StringVarP(&watchNotifyMethod, "notify-method", "n", "system", "Notification method: system, webhook, or both")
	watchCmd.Flags().BoolVarP(&watchQuiet, "quiet", "q", false, "Suppress output except errors")
}

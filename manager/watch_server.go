package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatih/color"
)

// WatchServerOptions contains configuration for the watch server
type WatchServerOptions struct {
	Port         int
	Interval     time.Duration
	NotifyMethod string
	Quiet        bool
}

// WatchServer monitors worktrees for conflicts
type WatchServer struct {
	wm      *WorktreeManager
	options WatchServerOptions
	server  *http.Server

	// State management
	mu               sync.RWMutex
	lastCheck        time.Time
	lastConflicts    []ConflictInfo
	currentConflicts []ConflictInfo
	checkCount       int
}

// WatchStatus represents the current status of the watch server
type WatchStatus struct {
	Running    bool           `json:"running"`
	LastCheck  time.Time      `json:"last_check"`
	NextCheck  time.Time      `json:"next_check"`
	CheckCount int            `json:"check_count"`
	Interval   string         `json:"interval"`
	Conflicts  []ConflictInfo `json:"conflicts"`
}

// NewWatchServer creates a new watch server instance
func NewWatchServer(wm *WorktreeManager, options WatchServerOptions) *WatchServer {
	return &WatchServer{
		wm:      wm,
		options: options,
	}
}

// Start starts the watch server
func (ws *WatchServer) Start(ctx context.Context) error {
	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/status", ws.handleStatus)
	mux.HandleFunc("/worktrees", ws.handleWorktrees)
	mux.HandleFunc("/conflicts", ws.handleConflicts)
	mux.HandleFunc("/check", ws.handleCheck)

	ws.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ws.options.Port),
		Handler: mux,
	}

	// Start the periodic checker
	go ws.runPeriodicCheck(ctx)

	// Start the HTTP server
	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			if !ws.options.Quiet {
				fmt.Printf("❌ HTTP server error: %v\n", err)
			}
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown the server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return ws.server.Shutdown(shutdownCtx)
}

// runPeriodicCheck runs the conflict check periodically
func (ws *WatchServer) runPeriodicCheck(ctx context.Context) {
	// Run initial check
	ws.performCheck()

	ticker := time.NewTicker(ws.options.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ws.performCheck()
		}
	}
}

// performCheck performs a conflict check
func (ws *WatchServer) performCheck() {
	ws.mu.Lock()
	ws.checkCount++
	checkNum := ws.checkCount
	ws.mu.Unlock()

	if !ws.options.Quiet {
		fmt.Printf("\n🔍 Running conflict check #%d at %s\n", checkNum, time.Now().Format("15:04:05"))
	}

	conflicts, err := ws.wm.CheckRebaseConflicts()
	if err != nil {
		if !ws.options.Quiet {
			fmt.Printf("❌ Error checking conflicts: %v\n", err)
		}
		return
	}

	ws.mu.Lock()
	ws.lastCheck = time.Now()
	ws.lastConflicts = ws.currentConflicts
	ws.currentConflicts = conflicts
	ws.mu.Unlock()

	// Check for new conflicts
	if HasNewConflicts(ws.lastConflicts, conflicts) {
		ws.notifyConflicts(conflicts)
	}

	if !ws.options.Quiet {
		if len(conflicts) == 0 {
			fmt.Printf("%s No conflicts detected\n", color.GreenString("✓"))
		} else {
			fmt.Printf("%s Found %d branches with potential conflicts\n", color.YellowString("⚠️"), len(conflicts))
			for _, c := range conflicts {
				if len(c.ConflictFiles) > 0 {
					fmt.Printf("  - %s: %d conflicting files\n", c.Branch, len(c.ConflictFiles))
				}
			}
		}
	}
}

// notifyConflicts sends notifications about conflicts
func (ws *WatchServer) notifyConflicts(conflicts []ConflictInfo) {
	// Check if notifications are enabled in config
	if ws.wm.Config != nil && ws.wm.Config.Watch != nil && !ws.wm.Config.Watch.NotifyOnConflicts {
		return
	}

	for _, conflict := range conflicts {
		if len(conflict.ConflictFiles) == 0 {
			continue
		}

		// Check if branch should be ignored
		if ws.shouldIgnoreBranch(conflict.Branch) {
			continue
		}

		// Build notification message
		message := fmt.Sprintf("⚠️ Workie: Branch '%s' would conflict rebasing on main", conflict.Branch)
		if len(conflict.ConflictFiles) > 0 {
			message += fmt.Sprintf(" (%d files)", len(conflict.ConflictFiles))
		}

		// Send notification based on method
		if ws.options.NotifyMethod == "system" || ws.options.NotifyMethod == "both" {
			input := &NotificationInput{
				Message:       message,
				HookEventName: "workie_watch_conflict",
			}

			if err := ws.wm.SendSystemNotification(input); err != nil {
				if !ws.options.Quiet {
					fmt.Printf("❌ Failed to send notification: %v\n", err)
				}
			}
		}

		// TODO: Add webhook support when NotifyMethod is "webhook" or "both"
	}
}

// shouldIgnoreBranch checks if a branch should be ignored based on config patterns
func (ws *WatchServer) shouldIgnoreBranch(branch string) bool {
	if ws.wm.Config == nil || ws.wm.Config.Watch == nil {
		return false
	}

	for _, pattern := range ws.wm.Config.Watch.BranchesToIgnore {
		// Simple glob pattern matching
		if matched, _ := filepath.Match(pattern, branch); matched {
			return true
		}
	}

	return false
}

// HTTP Handlers

func (ws *WatchServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws.mu.RLock()
	status := WatchStatus{
		Running:    true,
		LastCheck:  ws.lastCheck,
		NextCheck:  ws.lastCheck.Add(ws.options.Interval),
		CheckCount: ws.checkCount,
		Interval:   ws.options.Interval.String(),
		Conflicts:  ws.currentConflicts,
	}
	ws.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (ws *WatchServer) handleWorktrees(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	worktrees, err := ws.wm.GetWorktrees()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(worktrees)
}

func (ws *WatchServer) handleConflicts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws.mu.RLock()
	conflicts := ws.currentConflicts
	ws.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conflicts)
}

func (ws *WatchServer) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Run check in background
	go ws.performCheck()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "check initiated",
	})
}

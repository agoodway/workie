package autopilot

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PipelineOptions configures a pipeline run.
type PipelineOptions struct {
	Changes      []string
	StartPhase   Phase
	Backend      Backend
	Model        string
	Notify       bool
	AutoApprove  bool
	DryRun       bool
	NoCommit     bool
	ShowOutput   bool
	LogDir       string
	PhaseTimeout time.Duration

	OpencodeServerURL      string
	OpencodeServerUsername string
	OpencodeServerPassword string
}

// PhaseResult captures the outcome of a single phase execution.
type PhaseResult struct {
	Phase          PhaseInfo
	Success        bool
	StartedAt      time.Time
	Duration       time.Duration
	CommitDuration time.Duration
	LogFile        string
	Error          error
}

// ChangeResult captures the outcome of all phases for a single change.
type ChangeResult struct {
	Name     string
	Results  []PhaseResult
	Duration time.Duration
	Success  bool
}

// Pipeline orchestrates the full aviation pipeline for one or more changes.
type Pipeline struct {
	opts       PipelineOptions
	projectDir string
	logDir     string
	runner     SessionRunner
	notifier   Notifier
	debugLog   *log.Logger
	debugFile  *os.File
}

// NewPipeline creates a new pipeline with the given options.
func NewPipeline(opts PipelineOptions) *Pipeline {
	if opts.Backend == "" {
		opts.Backend = BackendClaude
	}
	if opts.Model == "" {
		opts.Model = DefaultModelForBackend(opts.Backend)
	}
	if opts.Backend == BackendOpencode {
		if strings.TrimSpace(opts.OpencodeServerURL) == "" {
			opts.OpencodeServerURL = DefaultOpencodeServerURL
		}
		if strings.TrimSpace(opts.OpencodeServerPassword) != "" && strings.TrimSpace(opts.OpencodeServerUsername) == "" {
			opts.OpencodeServerUsername = DefaultOpencodeServerUsername
		}
	}

	projectDir, err := os.Getwd()
	if err != nil {
		projectDir = "."
	}

	logDir := opts.LogDir
	if logDir == "" {
		logDir = filepath.Join(projectDir, ".autopilot-logs")
	}

	return &Pipeline{
		opts:       opts,
		projectDir: projectDir,
		logDir:     logDir,
		runner:     newSessionRunner(opts),
		notifier:   newNotifier(opts.Notify),
	}
}

// initDebugLog creates the pipeline debug log file.
func (p *Pipeline) initDebugLog() {
	if err := os.MkdirAll(p.logDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create log dir: %v\n", err)
		p.debugLog = log.New(io.Discard, "", 0)
		return
	}

	logPath := filepath.Join(p.logDir, "pipeline.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to open pipeline log: %v\n", err)
		p.debugLog = log.New(io.Discard, "", 0)
		return
	}

	p.debugFile = f
	p.debugLog = log.New(f, "", log.Ldate|log.Ltime|log.Lmicroseconds)
	p.logDebug("=== Pipeline started ===")
	p.logDebug("  project_dir=%s", p.projectDir)
	p.logDebug("  log_dir=%s", p.logDir)
	p.logDebug("  changes=%v", p.opts.Changes)
	p.logDebug("  backend=%s model=%s start_phase=%d", p.opts.Backend, p.opts.Model, p.opts.StartPhase)
	p.logDebug("  phase_timeout=%s", p.opts.PhaseTimeout)
	p.logDebug("  auto_approve=%v dry_run=%v no_commit=%v show_output=%v",
		p.opts.AutoApprove, p.opts.DryRun, p.opts.NoCommit, p.opts.ShowOutput)
}

// closeDebugLog closes the pipeline debug log file.
func (p *Pipeline) closeDebugLog() {
	if p.debugFile != nil {
		p.logDebug("=== Pipeline finished ===")
		p.debugFile.Close()
	}
}

// logDebug writes a formatted message to the pipeline debug log.
func (p *Pipeline) logDebug(format string, args ...any) {
	if p.debugLog != nil {
		p.debugLog.Printf(format, args...)
	}
}

func (p *Pipeline) notify(title, message string) {
	if p.notifier == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.notifier.Notify(ctx, title, message); err != nil {
		p.logDebug("notify: failed: %v", err)
		return
	}

	p.logDebug("notify: %s | %s", title, message)
}

// Run executes the full pipeline across all changes.
func (p *Pipeline) Run() error {
	if err := validateOptions(p.opts); err != nil {
		return err
	}

	// Validate all changes upfront
	for _, change := range p.opts.Changes {
		if err := validateChange(p.projectDir, change); err != nil {
			return err
		}
	}

	p.initDebugLog()
	defer p.closeDebugLog()
	p.printBanner()
	p.notify("Autopilot started", fmt.Sprintf("Running %d change(s)", len(p.opts.Changes)))

	if !p.opts.DryRun {
		if err := p.preflightCheck(); err != nil {
			p.notify("Preflight failed", err.Error())
			return err
		}
	}

	totalStart := time.Now()
	var changeResults []ChangeResult

	for i, change := range p.opts.Changes {
		startPhase := Phase(1)
		if i == 0 {
			startPhase = p.opts.StartPhase
		}

		p.logDebug("--- Change %d/%d: %s (start_phase=%d) ---", i+1, len(p.opts.Changes), change, startPhase)

		cr := p.runChange(change, startPhase, i)
		changeResults = append(changeResults, cr)

		if cr.Success {
			p.logDebug("change %q succeeded in %s", change, formatDuration(cr.Duration))
			fmt.Printf("\n\u2705 Change %q completed in %s\n", change, formatDuration(cr.Duration))
		} else {
			p.logDebug("change %q FAILED after %s", change, formatDuration(cr.Duration))
			fmt.Printf("\n\u274c Change %q failed after %s\n", change, formatDuration(cr.Duration))
		}

		p.printChangeTimings(cr)

		if !cr.Success {
			failedResult := cr.Results[len(cr.Results)-1]
			remaining := p.opts.Changes[i:]

			p.logDebug("aborting pipeline: failed at phase %d (%s), %d remaining changes skipped",
				failedResult.Phase.Number, failedResult.Phase.Name, len(remaining)-1)

			fmt.Printf("\nAborting pipeline: failed at phase %d (%s)", failedResult.Phase.Number, failedResult.Phase.Name)
			if len(remaining) > 1 {
				fmt.Printf(", %d remaining change(s) skipped", len(remaining)-1)
			}
			fmt.Println()

			resumeArgs := strings.Join(remaining, " ")
			fmt.Printf("\n   Resume with: workie autopilot %s --start-phase %d\n", resumeArgs, failedResult.Phase.Number)
			p.notify("Autopilot failed", fmt.Sprintf("%s failed at phase %d", change, failedResult.Phase.Number))

			p.printSummary(changeResults, time.Since(totalStart))
			return fmt.Errorf("change %q failed at phase %d (%s)", change, failedResult.Phase.Number, failedResult.Phase.Name)
		}
	}

	p.logDebug("all changes completed in %s", formatDuration(time.Since(totalStart)))
	p.notify("Autopilot completed", fmt.Sprintf("Finished %d change(s) in %s", len(changeResults), formatDuration(time.Since(totalStart))))
	p.printSummary(changeResults, time.Since(totalStart))
	return nil
}

// runChange runs all phases for a single change starting from startPhase.
// changeIndex is the position of this change in the pipeline (0-based).
func (p *Pipeline) runChange(name string, startPhase Phase, changeIndex int) ChangeResult {
	start := time.Now()
	phases := Phases()
	cr := ChangeResult{Name: name, Success: true}

	ctx := PromptContext{
		Change:      name,
		IsFirst:     changeIndex == 0,
		IsLast:      changeIndex == len(p.opts.Changes)-1,
		MultiChange: len(p.opts.Changes) > 1,
	}

	fmt.Printf("\n\u2708\ufe0f  Processing change: %s\n", name)
	if startPhase > 1 {
		fmt.Printf("   Resuming from phase %d\n", startPhase)
	}
	fmt.Println(strings.Repeat("\u2500", 50))
	p.notify("Change started", fmt.Sprintf("%s starting at phase %d", name, startPhase))

	for _, phase := range phases {
		if phase.Number < startPhase {
			continue
		}

		result := p.runPhase(ctx, phase)

		// Determine whether to run an intermediate commit.
		// Normally, phases with HandlesOwnCommit (land-the-plane) skip pipeline commits.
		// In multi-change mode, intermediate land-the-plane phases told the skill to skip
		// commit/PR, so the pipeline needs to make the commit instead.
		skipCommit := phase.HandlesOwnCommit
		if ctx.MultiChange && !ctx.IsLast && phase.Number == PhaseLandThePlane {
			skipCommit = false
		}

		if result.Success && !skipCommit && !p.opts.NoCommit {
			p.logDebug("[%s] phase %d: running intermediate commit", name, phase.Number)
			commitStart := time.Now()
			if err := p.runGitCommit(name, phase); err != nil {
				p.logDebug("[%s] phase %d: commit FAILED: %v", name, phase.Number, err)
				fmt.Printf("\u26a0\ufe0f  Intermediate commit failed: %v (continuing)\n", err)
			} else {
				p.logDebug("[%s] phase %d: commit completed in %s", name, phase.Number, formatDuration(time.Since(commitStart)))
			}
			result.CommitDuration = time.Since(commitStart)
		}

		cr.Results = append(cr.Results, result)

		if !result.Success {
			p.logDebug("[%s] phase %d FAILED after %s: %v", name, phase.Number, formatDuration(result.Duration), result.Error)
			fmt.Printf("\n\u274c Phase %d failed (%s)\n", phase.Number, formatDuration(result.Duration))
			if result.LogFile != "" {
				fmt.Printf("   Log: %s\n", result.LogFile)
			}
			if result.Error != nil {
				fmt.Printf("   Error: %v\n", result.Error)
			}
			fmt.Printf("\n   Resume with: workie autopilot %s --start-phase %d\n", name, phase.Number)
			p.notify(fmt.Sprintf("Phase %d failed", phase.Number), fmt.Sprintf("%s: %s", name, phase.Name))
			p.notify("Change failed", fmt.Sprintf("%s failed at phase %d", name, phase.Number))
			cr.Success = false
			cr.Duration = time.Since(start)
			return cr
		}

		p.logDebug("[%s] phase %d completed in %s", name, phase.Number, formatDuration(result.Duration))
		fmt.Printf("\u2705 Phase %d completed (%s)\n", phase.Number, formatDuration(result.Duration))
		p.notify(fmt.Sprintf("Phase %d completed", phase.Number), fmt.Sprintf("%s: %s in %s", name, phase.Name, formatDuration(result.Duration)))
	}

	cr.Duration = time.Since(start)
	p.notify("Change completed", fmt.Sprintf("%s finished in %s", name, formatDuration(cr.Duration)))
	return cr
}

// runPhase executes a single phase and returns the result.
func (p *Pipeline) runPhase(ctx PromptContext, info PhaseInfo) PhaseResult {
	header := fmt.Sprintf("[%d/5] %s %s \u2014 %s", info.Number, info.Icon, info.Name, info.Label)
	fmt.Printf("\n%s\n", header)

	p.logDebug("[%s] phase %d (%s) starting", ctx.Change, info.Number, info.Name)

	if p.opts.DryRun {
		prompt := buildPrompt(ctx, info)
		p.logDebug("[%s] phase %d: dry-run, prompt: %s", ctx.Change, info.Number, prompt)
		fmt.Printf("   [dry-run] would execute %s backend with prompt:\n   %s\n", p.opts.Backend, prompt)
		return PhaseResult{Phase: info, Success: true, StartedAt: time.Now()}
	}

	// Create log directory
	changeLogDir := filepath.Join(p.logDir, ctx.Change)
	if err := os.MkdirAll(changeLogDir, 0o755); err != nil {
		p.logDebug("[%s] phase %d: failed to create log dir: %v", ctx.Change, info.Number, err)
		return PhaseResult{Phase: info, Error: fmt.Errorf("failed to create log dir: %w", err)}
	}

	logFile := filepath.Join(changeLogDir, fmt.Sprintf("phase-%d-%s.log", info.Number, info.Name))
	prompt := buildPrompt(ctx, info)

	p.logDebug("[%s] phase %d: log_file=%s", ctx.Change, info.Number, logFile)
	p.logDebug("[%s] phase %d: prompt=%s", ctx.Change, info.Number, prompt)
	p.logDebug("[%s] phase %d: launching %s session", ctx.Change, info.Number, p.opts.Backend)

	timeoutCtx, cancel := context.WithTimeout(context.Background(), p.opts.PhaseTimeout)
	defer cancel()

	start := time.Now()
	err := p.executeSession(timeoutCtx, prompt, logFile)
	dur := time.Since(start)

	if err != nil {
		p.logDebug("[%s] phase %d: %s session FAILED after %s: %v", ctx.Change, info.Number, p.opts.Backend, formatDuration(dur), err)
	} else {
		p.logDebug("[%s] phase %d: %s session completed in %s", ctx.Change, info.Number, p.opts.Backend, formatDuration(dur))
	}

	return PhaseResult{
		Phase:     info,
		Success:   err == nil,
		StartedAt: start,
		Duration:  dur,
		LogFile:   logFile,
		Error:     err,
	}
}

// runGitCommit spawns a backend session for an intermediate git commit.
func (p *Pipeline) runGitCommit(change string, info PhaseInfo) error {
	if p.opts.DryRun {
		prompt := buildCommitPrompt(change, info)
		fmt.Printf("   [dry-run] would commit: %s\n", prompt)
		return nil
	}

	changeLogDir := filepath.Join(p.logDir, change)
	logFile := filepath.Join(changeLogDir, fmt.Sprintf("phase-%d-commit.log", info.Number))
	prompt := buildCommitPrompt(change, info)

	p.logDebug("[%s] commit phase %d: launching %s session", change, info.Number, p.opts.Backend)
	p.logDebug("[%s] commit phase %d: prompt=%s", change, info.Number, prompt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return p.executeSession(ctx, prompt, logFile)
}

// executeSession runs a backend session and writes output to the phase log.
func (p *Pipeline) executeSession(ctx context.Context, prompt, logFile string) error {
	p.logDebug("exec: backend=%s log_file=%s", p.opts.Backend, logFile)

	lf, err := os.Create(logFile)
	if err != nil {
		p.logDebug("exec: failed to create log file: %v", err)
		return fmt.Errorf("failed to create log file: %w", err)
	}
	defer lf.Close()

	stdout := io.Writer(lf)
	stderr := io.Writer(lf)
	if p.opts.ShowOutput {
		stdout = io.MultiWriter(lf, os.Stdout)
		stderr = io.MultiWriter(lf, os.Stderr)
	}

	req := SessionRequest{
		Prompt:      prompt,
		Model:       p.opts.Model,
		WorkingDir:  p.projectDir,
		AutoApprove: p.opts.AutoApprove,
	}

	return p.runner.Run(ctx, req, stdout, stderr)
}

// preflightCheck verifies the selected backend is reachable before running phases.
func (p *Pipeline) preflightCheck() error {
	p.logDebug("preflight: starting health check")
	fmt.Print("\nPreflight check... ")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := SessionRequest{
		Prompt:      "say ok",
		Model:       p.opts.Model,
		WorkingDir:  p.projectDir,
		AutoApprove: p.opts.AutoApprove,
	}
	if err := p.runner.Preflight(ctx, req); err != nil {
		p.logDebug("preflight: FAILED: %v", err)
		return err
	}

	p.logDebug("preflight: passed")
	fmt.Println("ok")
	return nil
}

// validateChange checks that the openspec change directory and tasks.md exist.
func validateChange(projectDir, name string) error {
	changeDir := filepath.Join(projectDir, "openspec", "changes", name)
	info, err := os.Stat(changeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("change directory not found: %s\n\nExpected: %s\nAvailable changes can be found in openspec/changes/", name, changeDir)
		}
		return fmt.Errorf("cannot access change directory %s: %w", name, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("change path is not a directory: %s", changeDir)
	}

	tasksFile := filepath.Join(changeDir, "tasks.md")
	if _, err := os.Stat(tasksFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("tasks.md not found for change %q\n\nExpected: %s", name, tasksFile)
		}
		return fmt.Errorf("cannot access tasks.md for change %q: %w", name, err)
	}

	return nil
}

// validateOptions checks that pipeline options are valid.
func validateOptions(opts PipelineOptions) error {
	if len(opts.Changes) == 0 {
		return fmt.Errorf("at least one change name is required")
	}
	if opts.StartPhase < 1 || opts.StartPhase > 5 {
		return fmt.Errorf("start-phase must be between 1 and 5, got %d", opts.StartPhase)
	}
	if !opts.Backend.IsValid() {
		return fmt.Errorf("invalid backend %q (must be one of: %s)", opts.Backend, SupportedBackendsList)
	}
	if opts.Model == "" {
		return fmt.Errorf("model cannot be empty")
	}
	return nil
}

// printBanner prints the pipeline configuration summary.
func (p *Pipeline) printBanner() {
	fmt.Println()
	fmt.Println(strings.Repeat("\u2550", 50))
	fmt.Println("\u2708\ufe0f  Workie Autopilot")
	fmt.Println(strings.Repeat("\u2550", 50))
	fmt.Printf("  Changes:     %s\n", strings.Join(p.opts.Changes, ", "))
	fmt.Printf("  Backend:     %s\n", p.opts.Backend)
	fmt.Printf("  Model:       %s\n", p.opts.Model)
	fmt.Printf("  Start phase: %d\n", p.opts.StartPhase)
	fmt.Printf("  Timeout:     %s\n", p.opts.PhaseTimeout)
	if p.opts.AutoApprove {
		fmt.Printf("  Auto-approve: yes\n")
	}
	if p.opts.NoCommit {
		fmt.Printf("  Commits:     skipped\n")
	}
	if p.opts.DryRun {
		fmt.Printf("  Mode:        DRY RUN\n")
	}
	if p.debugFile != nil {
		fmt.Printf("  Debug log:   %s\n", p.debugFile.Name())
	}
	fmt.Println(strings.Repeat("\u2550", 50))
}

// printChangeTimings prints a per-phase timing breakdown for a single change.
func (p *Pipeline) printChangeTimings(cr ChangeResult) {
	if len(cr.Results) == 0 {
		return
	}

	fmt.Printf("\n  Phase Timings \u2014 %s\n", cr.Name)
	fmt.Printf("  %s\n", strings.Repeat("\u2500", 48))
	fmt.Printf("  %-28s %8s %6s\n", "Phase", "Duration", "%")
	fmt.Printf("  %s\n", strings.Repeat("\u2500", 48))

	for _, r := range cr.Results {
		phaseDur := r.Duration + r.CommitDuration
		pct := 0.0
		if cr.Duration > 0 {
			pct = float64(phaseDur) / float64(cr.Duration) * 100
		}

		label := fmt.Sprintf("%d. %s", r.Phase.Number, r.Phase.Name)
		status := " "
		if !r.Success {
			status = "\u274c"
		}
		fmt.Printf("  %s%-27s %8s %5.0f%%\n", status, label, formatDuration(r.Duration), pct)

		if r.CommitDuration > 0 {
			commitPct := float64(r.CommitDuration) / float64(cr.Duration) * 100
			fmt.Printf("   %-27s %8s %5.0f%%\n", "\u2514\u2500 commit", formatDuration(r.CommitDuration), commitPct)
		}
	}

	fmt.Printf("  %s\n", strings.Repeat("\u2500", 48))
	fmt.Printf("  %-28s %8s\n", "Total", formatDuration(cr.Duration))
}

// printSummary prints the final results of the pipeline run.
func (p *Pipeline) printSummary(results []ChangeResult, total time.Duration) {
	var succeeded, failed int
	var failedChanges []string
	for _, cr := range results {
		if cr.Success {
			succeeded++
		} else {
			failed++
			failedChanges = append(failedChanges, cr.Name)
		}
	}

	fmt.Println()
	fmt.Println(strings.Repeat("\u2550", 50))
	fmt.Println("\U0001f4ca Summary")
	fmt.Println(strings.Repeat("\u2550", 50))
	fmt.Printf("  Succeeded: %d\n", succeeded)
	fmt.Printf("  Failed:    %d\n", failed)
	fmt.Printf("  Duration:  %s\n", formatDuration(total))
	if len(failedChanges) > 0 {
		fmt.Printf("\n  Failed changes:\n")
		for _, c := range failedChanges {
			fmt.Printf("    \u274c %s\n", c)
		}
	}

	// Aggregate phase-level stats across all changes
	type phaseStat struct {
		count   int
		total   time.Duration
		slowest time.Duration
		commitT time.Duration
	}
	stats := make(map[Phase]*phaseStat)

	for _, cr := range results {
		for _, r := range cr.Results {
			s, ok := stats[r.Phase.Number]
			if !ok {
				s = &phaseStat{}
				stats[r.Phase.Number] = s
			}
			s.count++
			s.total += r.Duration
			s.commitT += r.CommitDuration
			if r.Duration > s.slowest {
				s.slowest = r.Duration
			}
		}
	}

	if len(stats) > 0 {
		fmt.Printf("\n  Phase Stats (across %d change(s)):\n", len(results))
		fmt.Printf("  %s\n", strings.Repeat("\u2500", 48))
		fmt.Printf("  %-22s %7s %7s %7s\n", "Phase", "Avg", "Slowest", "Commit")
		fmt.Printf("  %s\n", strings.Repeat("\u2500", 48))

		for _, phase := range Phases() {
			s, ok := stats[phase.Number]
			if !ok {
				continue
			}
			avg := s.total / time.Duration(s.count)
			commitAvg := ""
			if s.commitT > 0 {
				commitAvg = formatDuration(s.commitT / time.Duration(s.count))
			} else {
				commitAvg = "-"
			}
			fmt.Printf("  %d. %-19s %7s %7s %7s\n",
				phase.Number, phase.Name,
				formatDuration(avg),
				formatDuration(s.slowest),
				commitAvg,
			)
		}
		fmt.Printf("  %s\n", strings.Repeat("\u2500", 48))
	}

	fmt.Println(strings.Repeat("\u2550", 50))
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

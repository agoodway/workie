package autopilot

import "fmt"

// Phase represents a numbered pipeline phase (1-5).
type Phase int

const (
	PhasePrepareForTakeoff Phase = 1
	PhaseTakeoff           Phase = 2
	PhasePreLandingCheck   Phase = 3
	PhasePrepareForLanding Phase = 4
	PhaseLandThePlane      Phase = 5
)

// PhaseInfo holds metadata for a pipeline phase.
type PhaseInfo struct {
	Number           Phase
	Name             string
	Label            string
	Icon             string
	CommitTemplate   string
	HandlesOwnCommit bool
}

// Phases returns all 5 pipeline phases in order.
func Phases() []PhaseInfo {
	return []PhaseInfo{
		{
			Number:         PhasePrepareForTakeoff,
			Name:           "prepare-for-takeoff",
			Label:          "Plan the work",
			Icon:           "\U0001f4cb", // 📋
			CommitTemplate: "chore(openspec): import tasks for %s",
		},
		{
			Number:         PhaseTakeoff,
			Name:           "takeoff",
			Label:          "Implement the work",
			Icon:           "\U0001f680", // 🚀
			CommitTemplate: "feat(%s): implement changes",
		},
		{
			Number:         PhasePreLandingCheck,
			Name:           "pre-landing-check",
			Label:          "Review the work",
			Icon:           "\U0001f50d", // 🔍
			CommitTemplate: "chore(%s): code review findings",
		},
		{
			Number:         PhasePrepareForLanding,
			Name:           "prepare-for-landing",
			Label:          "Fix review findings",
			Icon:           "\U0001f527", // 🔧
			CommitTemplate: "fix(%s): address review findings",
		},
		{
			Number:           PhaseLandThePlane,
			Name:             "land-the-plane",
			Label:            "Ship the work",
			Icon:             "\u2708\ufe0f", // ✈️
			HandlesOwnCommit: true,
		},
	}
}

// PhaseByNumber returns the PhaseInfo for a given phase number.
func PhaseByNumber(n Phase) PhaseInfo {
	for _, p := range Phases() {
		if p.Number == n {
			return p
		}
	}
	return PhaseInfo{}
}

// PromptContext provides positional information for multi-change pipeline runs.
type PromptContext struct {
	Change      string
	IsFirst     bool
	IsLast      bool
	MultiChange bool
}

// buildPrompt constructs the Claude prompt for a given change and phase.
// In multi-change mode, it appends override instructions to control branch
// and PR behavior so all changes accumulate on a single feature branch.
func buildPrompt(ctx PromptContext, phase PhaseInfo) string {
	dir := fmt.Sprintf("openspec/changes/%s/", ctx.Change)
	base := fmt.Sprintf("Do not ask me any questions — use your best judgment for any decisions.")

	var prompt string

	switch phase.Number {
	case PhasePrepareForTakeoff:
		prompt = fmt.Sprintf(
			"Run /prepare-for-takeoff for the change named %q located at %s. %s",
			ctx.Change, dir, base,
		)
		if ctx.MultiChange && !ctx.IsFirst {
			prompt += " You are already on the correct feature branch. Do NOT warn about not being on main — this is expected for a multi-change pipeline run."
		}
	case PhaseTakeoff:
		prompt = fmt.Sprintf(
			"Run /takeoff for the change named %q located at %s. %s",
			ctx.Change, dir, base,
		)
		if ctx.MultiChange && !ctx.IsFirst {
			prompt += " IMPORTANT: Do NOT create a new feature branch. You are already on the correct feature branch from a previous change in this pipeline. Stay on the current branch, find the epic, and begin implementation immediately."
		}
	case PhasePreLandingCheck:
		prompt = fmt.Sprintf(
			"Run /pre-landing-check for the change named %q located at %s. %s",
			ctx.Change, dir, base,
		)
	case PhasePrepareForLanding:
		prompt = fmt.Sprintf(
			"Run /prepare-for-landing for the change named %q located at %s. %s",
			ctx.Change, dir, base,
		)
	case PhaseLandThePlane:
		prompt = fmt.Sprintf(
			"Run /land-the-plane for the change named %q located at %s. %s",
			ctx.Change, dir, base,
		)
		if ctx.MultiChange && !ctx.IsLast {
			prompt += " IMPORTANT: This is an intermediate change in a multi-change pipeline. After verifying and approving all issues for this change, do NOT create a PR, do NOT run /git commit, and do NOT return to main. Stay on the current branch — the next change will continue here."
		}
	default:
		return ""
	}

	return prompt
}

// buildCommitPrompt constructs a prompt for an intermediate git commit after a phase.
func buildCommitPrompt(change string, phase PhaseInfo) string {
	msg := fmt.Sprintf(phase.CommitTemplate, change)
	return fmt.Sprintf(
		"Run /git commit with the message %q. Do not ask me any questions.",
		msg,
	)
}

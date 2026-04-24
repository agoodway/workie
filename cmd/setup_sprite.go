package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/agoodway/workie/manager"

	"github.com/spf13/cobra"
)

var (
	spriteName string
	spriteOrg  string
)

// setupSpriteCmd represents the setup-sprite command
var setupSpriteCmd = &cobra.Command{
	Use:   "setup-sprite",
	Short: "Run setup commands on a sprite VM from workie config",
	Long: `Run pre and post setup commands on a sprite.dev VM using the
sprite section from your .workie.yaml configuration.

Commands are executed via sprite exec on the target VM. Use pre_setup
for environment preparation (paths, config) and post_setup for
verification or finalization steps.

Configuration example in .workie.yaml:

  sprite:
    name: "my-dev-vm"
    org: "my-org"
    pre_setup:
      - "mkdir -p ~/go/bin"
      - "echo 'export PATH=$PATH:~/go/bin' >> ~/.bashrc"
    post_setup:
      - "which td && td --version"
      - "echo 'Setup complete'"`,
	Example: `  # Run setup commands using config defaults
  workie setup-sprite

  # Override sprite name
  workie setup-sprite --sprite my-other-vm

  # Override org
  workie setup-sprite --org different-org

  # Verbose output with timing details
  workie --verbose setup-sprite`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetupSprite()
	},
}

func runSetupSprite() error {
	// Verify sprite CLI is available
	fmt.Printf("Checking for sprite CLI...\n")
	spritePath, err := exec.LookPath("sprite")
	if err != nil {
		return fmt.Errorf("sprite CLI not found in PATH\n\nTo fix this:\n  • Install sprite from https://sprite.dev\n  • Ensure the sprite binary is in your PATH")
	}
	fmt.Printf("Found sprite CLI: %s\n", spritePath)

	// Create manager, detect repo, load config
	fmt.Printf("Detecting git repository...\n")
	opts := manager.Options{
		ConfigFile: configFile,
		Verbose:    verbose,
		Quiet:      quiet,
	}
	wm := manager.NewWithOptions(opts)

	if err := wm.DetectGitRepository(); err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}
	fmt.Printf("Repository found: %s\n", wm.RepoPath)

	fmt.Printf("Loading configuration...\n")
	if err := wm.LoadConfig(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	if wm.Config.LoadedFrom != "" {
		fmt.Printf("Config loaded from: %s\n", wm.Config.LoadedFrom)
	}

	// Validate sprite config exists
	if wm.Config.Sprite == nil {
		return fmt.Errorf("no sprite section found in configuration\n\nTo fix this:\n  • Add a sprite section to your .workie.yaml\n  • Run 'workie init' to see the config template")
	}

	sprite := wm.Config.Sprite

	if len(sprite.PreSetup) == 0 && len(sprite.PostSetup) == 0 {
		return fmt.Errorf("no setup commands configured in sprite section\n\nTo fix this:\n  • Add pre_setup or post_setup commands to the sprite section in .workie.yaml")
	}

	// Resolve sprite name (flag overrides config)
	name := sprite.Name
	if spriteName != "" {
		fmt.Printf("Sprite name overridden by flag: %s -> %s\n", sprite.Name, spriteName)
		name = spriteName
	}
	if name == "" {
		return fmt.Errorf("sprite name is required\n\nTo fix this:\n  • Set sprite.name in .workie.yaml\n  • Or use --sprite flag to specify the name")
	}

	// Resolve org (flag overrides config)
	org := sprite.Org
	if spriteOrg != "" {
		fmt.Printf("Org overridden by flag: %s -> %s\n", sprite.Org, spriteOrg)
		org = spriteOrg
	}

	fmt.Printf("\n--- Sprite Setup ---\n")
	fmt.Printf("VM name:    %s\n", name)
	if org != "" {
		fmt.Printf("Org:        %s\n", org)
	}
	fmt.Printf("Pre-setup:  %d command(s)\n", len(sprite.PreSetup))
	fmt.Printf("Post-setup: %d command(s)\n", len(sprite.PostSetup))
	fmt.Printf("--------------------\n\n")

	// Check sprite VM exists (warn if not)
	fmt.Printf("Checking if sprite VM '%s' exists...\n", name)
	if err := checkSpriteExists(name, org); err != nil {
		return err
	}

	totalStart := time.Now()

	// Run pre-setup commands if configured
	if len(sprite.PreSetup) > 0 {
		if err := runSpriteCommands(name, org, sprite.PreSetup, "pre_setup"); err != nil {
			return fmt.Errorf("pre_setup failed: %w", err)
		}
		fmt.Println()
	}

	// Run post-setup commands if configured
	if len(sprite.PostSetup) > 0 {
		if err := runSpriteCommands(name, org, sprite.PostSetup, "post_setup"); err != nil {
			return fmt.Errorf("post_setup failed: %w", err)
		}
	}

	// Print summary
	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Total time: %s\n", time.Since(totalStart).Round(time.Millisecond))
	if len(sprite.PreSetup) > 0 {
		fmt.Printf("Pre-setup:  %d command(s) ran\n", len(sprite.PreSetup))
	}
	if len(sprite.PostSetup) > 0 {
		fmt.Printf("Post-setup: %d command(s) ran\n", len(sprite.PostSetup))
	}
	fmt.Printf("---------------\n")

	return nil
}

func runSpriteCommands(name, org string, commands []string, phase string) error {
	fmt.Printf("Running %s commands (%d)...\n", phase, len(commands))

	for i, cmd := range commands {
		fmt.Printf("[%d/%d] %s\n", i+1, len(commands), cmd)

		args := []string{"exec", "-s", name}
		if org != "" {
			args = append(args, "-o", org)
		}
		args = append(args, "--", "sh", "-c", cmd)

		fmt.Printf("  Running: sprite %s\n", strings.Join(args, " "))

		start := time.Now()

		execCmd := exec.Command("sprite", args...)
		var stdout, stderr strings.Builder
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr

		err := execCmd.Run()
		elapsed := time.Since(start).Round(time.Millisecond)

		if err != nil {
			fmt.Printf("  FAILED in %s\n", elapsed)
			if stderr.Len() > 0 {
				fmt.Printf("  Stderr: %s\n", strings.TrimSpace(stderr.String()))
			}
			if stdout.Len() > 0 {
				fmt.Printf("  Stdout: %s\n", strings.TrimSpace(stdout.String()))
			}
			return fmt.Errorf("%s command [%d/%d] failed: %w", phase, i+1, len(commands), err)
		}

		fmt.Printf("  OK in %s\n", elapsed)
		if stdout.Len() > 0 {
			fmt.Printf("  Output: %s\n", strings.TrimSpace(stdout.String()))
		}
		if stderr.Len() > 0 {
			fmt.Printf("  Stderr: %s\n", strings.TrimSpace(stderr.String()))
		}
	}

	return nil
}

func checkSpriteExists(name, org string) error {
	args := []string{"list"}
	if org != "" {
		args = append(args, "-o", org)
	}

	fmt.Printf("Running: sprite %s\n", strings.Join(args, " "))
	cmd := exec.Command("sprite", args...)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list sprites: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == name {
			fmt.Printf("Sprite VM '%s' found\n", name)
			return nil
		}
	}

	// Build the create command for the warning message
	createCmd := fmt.Sprintf("sprite create %s", name)
	if org != "" {
		createCmd += fmt.Sprintf(" -o %s", org)
	}

	return fmt.Errorf("sprite VM '%s' does not exist\n\nTo fix this:\n  • Create it with: %s", name, createCmd)
}

func init() {
	rootCmd.AddCommand(setupSpriteCmd)

	setupSpriteCmd.Flags().StringVarP(&spriteName, "sprite", "s", "", "Override sprite VM name from config")
	setupSpriteCmd.Flags().StringVarP(&spriteOrg, "org", "o", "", "Override organization from config")
}

package main

import (
	"fmt"
	"os"
	"strconv"

	"mytask/internal/command"
	"mytask/internal/config"
	"mytask/internal/presets"
	"mytask/internal/task"

	"github.com/spf13/cobra"
)

var (
	store *task.Store
	cfg   *config.Config
	deps  *command.Dependencies
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "mytask",
	Short: "A task management tool",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		store, err = task.NewStore(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}
		deps = &command.Dependencies{
			Store:    store,
			HooksDir: cfg.HooksDir,
			Stdout:   os.Stdout,
			Stderr:   os.Stderr,
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if store != nil {
			store.Close()
		}
	},
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		desc, _ := cmd.Flags().GetString("description")

		t, err := command.Create(deps, name, desc)
		if err != nil {
			return err
		}

		fmt.Printf("Task created: ID=%d, Name=%s\n", t.ID, t.Name)
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return command.List(deps)
	},
}

var getCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get task details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}
		return command.Get(deps, id)
	},
}

var startCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start working on a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}

		t, err := command.Start(deps, id)
		if err != nil {
			return err
		}

		fmt.Printf("Task %d started, status: %s\n", t.ID, t.Status)
		return nil
	},
}

var submitCmd = &cobra.Command{
	Use:   "submit <id>",
	Short: "Submit a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}

		t, err := command.Submit(deps, id)
		if err != nil {
			return err
		}

		fmt.Printf("Task %d submitted, status: %s\n", t.ID, t.Status)
		return nil
	},
}

var completeCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Complete a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}

		t, err := command.Complete(deps, id)
		if err != nil {
			return err
		}

		fmt.Printf("Task %d completed\n", t.ID)
		return nil
	},
}

var resetCmd = &cobra.Command{
	Use:   "reset <id>",
	Short: "Skip a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID(args[0])
		if err != nil {
			return err
		}

		reason, _ := cmd.Flags().GetString("reason")

		t, err := command.Reset(deps, id, reason)
		if err != nil {
			return err
		}

		fmt.Printf("Task %d skipped, reason: %s\n", t.ID, reason)
		return nil
	},
}

var installHooksCmd = &cobra.Command{
	Use:   "install-hooks <hook-name>",
	Short: "Install a predefined hook",
	Long: `Install a predefined hook to the hooks directory.

Available hooks:
  git-reset   - Reset git state before operation
  git-commit  - Commit changes after operation
  notify      - Send desktop notification
  log         - Log operations to a file

Examples:
  mytask install-hooks git-commit --hook-type postcreate
  mytask install-hooks git-reset --hook-type prestart
  mytask install-hooks notify --hook-type poststart
  mytask install-hooks log --hook-type postsubmit`,
	Args: cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		hookName := args[0]
		hookType, _ := cmd.Flags().GetString("hook-type")
		force, _ := cmd.Flags().GetBool("force")

		scriptPath, err := presets.InstallHook(cfg.HooksDir, hookName, hookType, force)
		if err != nil {
			return err
		}

		fmt.Printf("Hook installed: %s -> %s\n", hookName, scriptPath)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Skip store cleanup
	},
}

var skillCmd = &cobra.Command{
	Use:   "skill <directory>",
	Short: "Generate a skill file for mytask",
	Args:  cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return nil // Skip config and store initialization
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		skillName, _ := cmd.Flags().GetString("name")
		baseDir := args[0]

		skillPath, err := presets.GenerateSkillFile(baseDir, skillName)
		if err != nil {
			return err
		}

		fmt.Printf("Skill file created: %s\n", skillPath)
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// Skip store cleanup
	},
}

func parseID(s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid task id: %s", s)
	}
	return id, nil
}

func init() {
	createCmd.Flags().StringP("name", "n", "", "task name")
	createCmd.Flags().StringP("description", "d", "", "task description")
	resetCmd.Flags().StringP("reason", "r", "", "reset reason")
	skillCmd.Flags().StringP("name", "n", "mytask", "skill name")
	installHooksCmd.Flags().String("hook-type", "", "hook type")
	installHooksCmd.Flags().BoolP("force", "f", false, "overwrite existing hook")

	rootCmd.AddCommand(createCmd, listCmd, getCmd, startCmd, submitCmd, completeCmd, resetCmd, skillCmd, installHooksCmd)
}
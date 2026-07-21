package main

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/lewtec/eletrocromo"
	"github.com/lucasew/contapila-go/internal/engine"
	"github.com/lucasew/contapila-go/internal/web"
	"github.com/lucasew/contapila-go/pkg/project"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// eletrocromoAppID is the reverse-domain Helium profile for contapila desktop.
const eletrocromoAppID = "br.tec.lew.contapila"

func desktopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "desktop [ledger]",
		Short: "Read-only UI in a Helium window (eletrocromo)",
		Long: `Open the same read-only web UI as "contapila web" inside a Helium
--app window via eletrocromo. The library owns loopback bind and token auth;
there is no --addr flag.

Project root is discovered from -C / the process working directory (walk up
for contapila.cue), same as other commands.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := projectCwd()
			if err != nil {
				return err
			}
			p, pdb, _, err := engine.OpenProject(cwd)
			if err != nil {
				return err
			}
			s, err := web.New(p, pdb)
			if err != nil {
				return err
			}
			// Optional ledger matches web: accepted for CLI parity; web only
			// prints a deep-link URL, desktop has no stdout banner.
			_ = args

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			app := eletrocromo.App{
				ID:      eletrocromoAppID,
				Handler: s.Handler(),
				Context: ctx,
			}
			return app.Run()
		},
	}
}

// applyDesktopRewrite mutates os.Args (and workDir when a project path is given)
// so bare not-a-TTY launches become "contapila desktop". See SPEC §3.2.1.
func applyDesktopRewrite() {
	stdinTTY := isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
	stdoutTTY := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	newArgs, setDir, ok := planDesktopRewrite(stdinTTY, stdoutTTY, os.Args[1:])
	if !ok {
		return
	}
	if setDir != "" {
		workDir = setDir
	}
	os.Args = append([]string{os.Args[0]}, newArgs...)
}

// planDesktopRewrite decides whether a not-a-TTY invocation should become desktop.
// args is os.Args[1:] (no program name). When ok, newArgs is the rewritten argv
// without the program name; setWorkDir is non-empty when the sole positional was
// a project directory or contapila.cue path.
func planDesktopRewrite(stdinTTY, stdoutTTY bool, args []string) (newArgs []string, setWorkDir string, ok bool) {
	if stdinTTY || stdoutTTY {
		return nil, "", false
	}

	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-v" || a == "--verbose":
			flags = append(flags, a)
		case a == "-C" || a == "--directory":
			if i+1 >= len(args) {
				// Incomplete flag — leave for cobra.
				return nil, "", false
			}
			flags = append(flags, a, args[i+1])
			i++
		case strings.HasPrefix(a, "--directory="):
			flags = append(flags, a)
		case strings.HasPrefix(a, "-"):
			// Unknown global flag or other command flag — do not rewrite.
			return nil, "", false
		default:
			positionals = append(positionals, a)
		}
	}

	switch len(positionals) {
	case 0:
		return append(flags, "desktop"), "", true
	case 1:
		dir, resolved := resolveProjectStartArg(positionals[0])
		if !resolved {
			return nil, "", false
		}
		return append(flags, "desktop"), dir, true
	default:
		return nil, "", false
	}
}

// resolveProjectStartArg maps a path positional to a project search start directory.
// Accepts an existing directory, or a file named contapila.cue (returns its parent).
func resolveProjectStartArg(arg string) (dir string, ok bool) {
	if arg == "" {
		return "", false
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		return abs, true
	}
	if filepath.Base(abs) == project.ProjectMarker {
		return filepath.Dir(abs), true
	}
	return "", false
}

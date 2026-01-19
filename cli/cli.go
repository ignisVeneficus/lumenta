package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ignisVeneficus/lumenta/config"
	"github.com/ignisVeneficus/lumenta/pipeline"
)

func Run(cfg config.Config) error {
	if len(os.Args) == 1 {
		return runServe(cfg)
	}

	cmd := os.Args[1]

	switch cmd {
	case "serve":
		return runServe(cfg)

	case "rebuild":
		return runRebuild(cfg, os.Args[2:])

	case "sync":
		return runSync(cfg, os.Args[2:])

	case "export":
		return runExport(cfg, os.Args[2:])

	case "import":
		return runImport(cfg, os.Args[2:])
	case "-h", "--help", "help":
		printGlobalHelp()
		return nil

	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

func printGlobalHelp() {
	fmt.Printf(`Usage: %s <command> [options]

Commands:
  sync        Synchronize filesystem with database
  rebuild     Rebuild albums and metadata
  status      Show current state

Use "%s <command> --help" for command-specific options.
`, os.Args[0], os.Args[0])
}

func runServe(cfg config.Config) error {
	return nil
}

func runRebuild(cfg config.Config, args []string) error {
	return nil
}

func runSync(cfg config.Config, args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)

	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s sync [options]\n\n", os.Args[0])
		fmt.Fprintln(fs.Output(), "Options:")
		fs.PrintDefaults()
	}

	noCleanUp := fs.Bool("noCleanUp", false, "do not delete images missing from sync result")
	fs.BoolVar(noCleanUp, "nc", false, "shorthand for -noCleanUp")

	cleanUp := true
	if noCleanUp != nil && (*noCleanUp) == true {
		cleanUp = false
	}

	if err := fs.Parse(args); err != nil {
		return err
	}
	err := pipeline.RunGlobalSync(context.Background(), cfg, cleanUp)
	return err
}
func runExport(cfg config.Config, args []string) error {
	return nil
}
func runImport(cfg config.Config, args []string) error {
	return nil
}

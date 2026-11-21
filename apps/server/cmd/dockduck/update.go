package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"dockduck/internal/updater"
)

// runUpdate is `dockduck update [--check]`: look for a newer release and install it.
// A running server is not touched; it keeps the old code until restarted
// (`systemctl restart dockduck`), or use Settings, which restarts it for you.
func runUpdate(args []string, out, errOut io.Writer) int {
	checkOnly := false
	for _, a := range args {
		switch a {
		case "--check":
			checkOnly = true
		case "--help", "-h":
			fmt.Fprintln(out, "usage: dockduck update [--check]\n\nInstalls the latest release from github.com/"+updater.Repo+" over this binary\n(checksum verified). --check only reports whether a newer version exists.")
			return 0
		default:
			fmt.Fprintf(errOut, "update: unknown argument %q\n", a)
			return 2
		}
	}
	u := &updater.Updater{Version: version}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	st, err := u.Check(ctx)
	if err != nil {
		fmt.Fprintln(errOut, "update:", err)
		return 1
	}
	if !st.Available {
		fmt.Fprintf(out, "dockduck %s is up to date (latest: %s)\n", st.Current, st.Latest)
		return 0
	}
	fmt.Fprintf(out, "new version: %s -> %s\n%s\n", st.Current, st.Latest, st.URL)
	if checkOnly {
		return 0
	}
	if !st.CanInstall {
		fmt.Fprintln(errOut, "update:", st.Reason)
		return 1
	}
	tag, err := u.Install(ctx)
	if err != nil {
		fmt.Fprintln(errOut, "update:", err)
		return 1
	}
	exe, _ := os.Executable()
	fmt.Fprintf(out, "installed %s at %s\n", tag, exe)
	if updater.Managed() {
		fmt.Fprintln(out, "restart the service to run it: systemctl restart dockduck")
	} else {
		fmt.Fprintln(out, "if DockDuck is running, restart it (or update from Settings next time, which restarts it for you)")
	}
	return 0
}

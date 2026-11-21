package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"dockduck/internal/auth"
	"dockduck/internal/config"
)

// runPasswd is `dockduck passwd [--config f]`: set or change the password from the
// machine itself. It is also the recovery path when the password is forgotten, so
// it does not ask for the old one: whoever can write the config file owns the box.
// A running server notices the change within a second and ends every session.
func runPasswd(args []string, in *os.File, out, errOut io.Writer) int {
	cfgPath := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--config", "-config":
			if i+1 >= len(args) {
				fmt.Fprintln(errOut, "passwd: --config needs a file")
				return 2
			}
			cfgPath = args[i+1]
			i++
		case "--help", "-h":
			fmt.Fprintln(out, "usage: dockduck passwd [--config FILE]\n\nSets (or resets) the DockDuck login password. The password is read from the\nterminal, or from stdin when it is piped (the first line).")
			return 0
		default:
			fmt.Fprintf(errOut, "passwd: unknown argument %q\n", args[i])
			return 2
		}
	}
	if cfgPath == "" {
		p, err := config.DefaultPath()
		if err != nil {
			fmt.Fprintln(errOut, "passwd:", err)
			return 1
		}
		cfgPath = p
	}
	store, err := config.Open(cfgPath)
	if err != nil {
		fmt.Fprintf(errOut, "passwd: config %s: %v\n", cfgPath, err)
		return 1
	}

	pw, err := readNewPassword(in, out)
	if err != nil {
		fmt.Fprintln(errOut, "passwd:", err)
		return 1
	}
	if err := auth.CheckPolicy(pw); err != nil {
		fmt.Fprintln(errOut, "passwd:", strings.TrimPrefix(err.Error(), "password policy: "))
		return 1
	}
	hash, err := auth.Hash(pw)
	if err != nil {
		fmt.Fprintln(errOut, "passwd:", err)
		return 1
	}
	if err := store.Update(func(c *config.Config) error { c.Auth.PasswordHash = hash; return nil }); err != nil {
		fmt.Fprintln(errOut, "passwd:", err)
		return 1
	}
	fmt.Fprintf(out, "Password saved to %s. Running servers pick it up within a second and end every login session.\n", cfgPath)
	return 0
}

// readNewPassword prompts twice on a terminal (no echo), or reads one line from a pipe.
func readNewPassword(in *os.File, out io.Writer) (string, error) {
	if term.IsTerminal(int(in.Fd())) {
		fmt.Fprint(out, "New password: ")
		a, err := term.ReadPassword(int(in.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return "", err
		}
		fmt.Fprint(out, "Repeat it: ")
		b, err := term.ReadPassword(int(in.Fd()))
		fmt.Fprintln(out)
		if err != nil {
			return "", err
		}
		if string(a) != string(b) {
			return "", errors.New("the two passwords differ")
		}
		return string(a), nil
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && line == "" {
		return "", errors.New("no password on stdin")
	}
	return strings.TrimRight(line, "\r\n"), nil
}

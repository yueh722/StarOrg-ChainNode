package lib

import (
	log "chain/lib/zlog"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/pkg/errors"
)

var (
	DefaultPropagateEnvironment = []string{"PATH", "TMPDIR"}
)

type Launcher struct {
	Name        string
	AppFilePath string
	Args        map[string]string
	Envs        []string
	Logger      *log.Logger
	NotifyFunc  NotifyFunc
}

// Sample
func (b *Launcher) sample() bool {

	filePath := ""
	cmd := b.NewCommand(filePath, "")
	err := b.RunCommand(cmd)
	if err != nil {
		return false
	}

	return true
}

// RunApp starts an execute file and returns a Session that can be used to
// signal it and wait for termination.
func (b *Launcher) RunApp() (*Session, error) {

	cmd := b.NewCommandFromMap(b.AppFilePath, b.Args)

	cmd.Dir = filepath.Dir(b.AppFilePath)

	sess, err := Start(b.Logger, cmd, func(err error) {
		b.NotifyFunc(true, err)
	})
	if err != nil {
		return nil, errors.Wrapf(err, "launcher '%s' failed to start", b.Name)
	}

	b.NotifyFunc(false, nil)

	return sess, nil
}

func (b *Launcher) NewCommandFromMap(name string, args map[string]string) *exec.Cmd {
	argStrings := []string{}

	flags := maputil.Keys(args)
	sort.Strings(flags)

	//iterator args
	for _, arg := range flags {
		argStrings = append(argStrings, arg)
		if args[arg] != "" {
			argStrings = append(argStrings, args[arg])
		}
	}

	log.Infof("cmd: %s %s", name, strings.Join(argStrings, " "))

	return b.NewCommand(name, argStrings...)
}

// NewCommand creates an exec.Cmd that is configured to prune the calling
// environment down to the environment variables specified in the
// Launcher's Environment and the DefaultPropagateEnvironment.
func (b *Launcher) NewCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	propagationList := appendDefaultPropagateEnvironment(b.Envs)
	for _, key := range propagationList {
		if val, ok := os.LookupEnv(key); ok {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, val))
		}
	}
	return cmd
}

// runCommand runs a command and waits for it to complete.
func (b *Launcher) RunCommand(cmd *exec.Cmd) error {
	sess, err := Start(b.Logger, cmd)
	if err != nil {
		return err
	}
	return sess.Wait()
}

func appendDefaultPropagateEnvironment(propagateEnvironment []string) []string {
	if propagateEnvironment == nil || len(propagateEnvironment) == 0 {
		return []string{}
	}
	for _, variable := range DefaultPropagateEnvironment {
		if !contains(propagateEnvironment, variable) {
			propagateEnvironment = append(propagateEnvironment, variable)
		}
	}
	return propagateEnvironment
}

func contains(propagateEnvironment []string, key string) bool {
	for _, variable := range propagateEnvironment {
		if key == variable {
			return true
		}
	}
	return false
}

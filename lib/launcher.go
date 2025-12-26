package lib

import (
	"chain/lib/log"
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
)

var (
	DefaultPropagateEnvironment = []string{"NODECLI_HOME", "PATH", "TMPDIR"}
)

type Launcher struct {
	Name        string
	AppFilePath string
	Args        map[string]string
	Envs        []string
	Logger      *log.Logger
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
	sess, err := Start(b.Logger, cmd, func(error) {})
	if err != nil {
		return nil, errors.Wrapf(err, "launcher '%s' failed to start", b.Name)
	}

	return sess, nil

	/*


		// run app
		// start abci-v2app --home .\home0
		//	--socket-addr tcp://127.0.0.1:26658
		//	--http-addr 127.0.0.1:8080
		//	--abci-rpc-addr http://127.0.0.1:26657

		args := map[string]string{
			"home":          appDir,
			"--socket-addr":   "0.0.0.0",
			"--http-addr":     "0.0.0.0",
			"--abci-rpc-addr": "0.0.0.0",
		}

		run1 := filepath.Join("./bin", "app.exe")
		cmd1 := b.NewCommandFromMap(run1, args)
		sess1, err1 := Start(b.Logger, cmd1, func(error) {})
		if err1 != nil {
			return nil, nil, errors.Wrapf(err1, "launcher '%s' app failed to start", b.Name)
		}

		return sess0, sess1, nil
	*/
}

func (b *Launcher) NewCommandFromMap(name string, args map[string]string) *exec.Cmd {

	argString := ""

	//iterator args
	for arg, val := range args {
		argString += " " + fmt.Sprintf("%s %s", arg, val)
	}

	return b.NewCommand(name, argString)
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

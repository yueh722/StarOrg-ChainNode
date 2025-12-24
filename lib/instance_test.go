package lib

import (
	"Chain/lib/log"
	"Chain/lib/queuestore"
	"Chain/utils"
	"testing"
	"time"
)

func TestInstance_StartNode(t *testing.T) {

	core, err := queuestore.NewSQLiteStore(
		queuestore.SQLiteConfig{
			Path:        utils.GetLogPath("nodeproxy", "node01"),
			Migrate:     true,
			OpTimeout:   5 * time.Second,
			InitTimeout: 3 * time.Second,
		})
	if err != nil {
		t.Fatal(err)
	}

	instance := NewInstance(
		&Launcher{
			Name:        "Start-Cometbft-Node-id",
			AppFilePath: "./bin/cometbft.exe",
			Args: map[string]string{
				"node":   "",
				"--home": "./node01",
			},
			Envs:   nil,
			Logger: log.NewCometbftNodeLogger("node01", core),
		},
	)

	if err := instance.Start(); err != nil {
		t.Error(err)
	}

	instance.Wait()

	time.Sleep(100 * time.Millisecond)

	instance.Stop()
}

func TestInstance_StartApp(t *testing.T) {
	instance := NewInstance(
		&Launcher{
			Name:        "Start-Cometbft-App",
			AppFilePath: "./bin/abci-v2-Identity-app.exe",
			Args: map[string]string{
				"--home":          " ./app01",
				"--socket-addr":   "tcp://127.0.0.1:26658",
				"--http-addr":     "127.0.0.1:8080",
				"--abci-rpc-addr": "http://127.0.0.1:26657",
			},
			Envs:   nil,
			Logger: nil,
		},
	)

	if err := instance.Start(); err != nil {
		t.Error(err)
	}

	instance.Wait()

	time.Sleep(100 * time.Millisecond)

	instance.Stop()
}

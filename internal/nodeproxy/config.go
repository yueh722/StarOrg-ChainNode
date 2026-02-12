package nodeproxy

import (
	"chain/internal/common"
	"chain/lib/zlog"
	"chain/utils"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/samber/lo"
	_ "github.com/spf13/viper/remote"
	"gopkg.in/ini.v1"
)

const (
	SingleModel = "single"
	MultiModel  = "multi"
)

type Config struct {
	AppName    string
	ServerAddr string
	DeviceID   string
	HomeDir    string
	BinDir     string // if leave blank, HomeDir/Bin will be applied.
}

type GlobalConfig struct {
	ServerAddr    string
	DeviceID      string
	HomeDir       string
	BinDir        string // if leave blank, HomeDir/Bin will be applied.
	LauncherModel string // if leave blank, use Single
}

func (cfg GlobalConfig) DefaultHomeDir() string {
	return common.NodeProxyAppDataPath
}

type AppConfig struct {
	AppName string
	HomeDir string
}

var cfgMux = sync.Mutex{}

func InitializeDeviceID(cfgFiles []string) error {
	if len(cfgFiles) == 0 {
		return errors.New("no config files specified")
	}

	for _, cfgFile := range cfgFiles {
		if !fileutil.IsExist(cfgFile) {
			continue
		}

		iniFile, err := ini.LoadSources(ini.LoadOptions{Insensitive: true}, cfgFile)
		if err != nil {
			return fmt.Errorf("read config file error: %v", err)
		}

		// global config
		globalCfg := &GlobalConfig{}
		if err := iniFile.Section("global").MapTo(globalCfg); err != nil {
			return fmt.Errorf("data in config file error: %v", err)
		}

		if globalCfg.DeviceID == "" {
			globalCfg.DeviceID = utils.DeviceID()
			iniFile.Section("global").Key("DeviceId").SetValue(globalCfg.DeviceID)
			if err := iniFile.SaveTo(cfgFile); err != nil {
				return fmt.Errorf("save deviceID to config file of NodeProxy error: %v", err)
			}

			zlog.Infof("create a new device ID %v with current config", globalCfg.DeviceID)
		} else {
			zlog.Info("loaded device ID ", globalCfg.DeviceID)
		}

		return nil
	}

	return fmt.Errorf("no config files found from: %v", cfgFiles)
}

func GetManagedApps(cfgFiles []string) ([]AppConfig, string, error) {
	if len(cfgFiles) == 0 {
		return nil, "", errors.New("no config files specified")
	}

	for _, cfgFile := range cfgFiles {
		if !fileutil.IsExist(cfgFile) {
			continue
		}

		iniFile, err := ini.LoadSources(ini.LoadOptions{Insensitive: true}, cfgFile)
		if err != nil {
			return nil, "", fmt.Errorf("read config file error: %v", err)
		}

		sections := iniFile.SectionStrings()
		sections = slice.Filter(sections, func(_ int, sectionName string) bool {
			return strings.HasPrefix(sectionName, "node")
		})

		sort.Strings(sections)

		return slice.Map(sections, func(_ int, sectionName string) AppConfig {
			homeDirs := []string{
				iniFile.Section(sectionName).Key("HomeDir").String(),
				iniFile.Section("global").Key("HomeDir").String(),
				GlobalConfig{}.DefaultHomeDir(),
			}

			homeDir, _ := lo.Find(homeDirs, func(str string) bool {
				return str != ""
			})

			appName := sectionName
			//appName := iniFile.Section(sectionName).Key("AppName").String()

			return AppConfig{
				AppName: appName,
				HomeDir: filepath.Join(homeDir, appName),
			}
		}), cfgFile, nil
	}

	return nil, "", fmt.Errorf("no config files found from: %v", cfgFiles)
}

func LoadGlobalConfig(cfgFiles []string) (*GlobalConfig, error) {

	return nil, nil
}

func LoadConfig(cfgFiles []string, appName string) (*Config, error) {

	cfgMux.Lock()
	defer cfgMux.Unlock()

	if len(cfgFiles) == 0 {
		return nil, errors.New("no config files specified")
	}

	for _, cfgFile := range cfgFiles {
		if !fileutil.IsExist(cfgFile) {
			continue
		}

		iniFile, err := ini.LoadSources(ini.LoadOptions{Insensitive: true}, cfgFile)
		if err != nil {
			return nil, fmt.Errorf("read config file error: %v", err)
		}

		globalCfg := &GlobalConfig{}
		if err := iniFile.Section("global").MapTo(globalCfg); err != nil {
			return nil, fmt.Errorf("data in config file error: %v", err)
		}

		if globalCfg.HomeDir == "" {
			globalCfg.HomeDir = globalCfg.DefaultHomeDir()
		}

		if globalCfg.LauncherModel == "" {
			globalCfg.LauncherModel = MultiModel
		}

		if globalCfg.BinDir == "" {
			globalCfg.BinDir = filepath.Join(globalCfg.HomeDir, "bin")
		}

		cfg := &Config{}
		if err := iniFile.Section(appName).MapTo(cfg); err != nil {
			return nil, fmt.Errorf("could not find app section '%v', error: %v", appName, err)
		}

		cfg.AppName = appName
		cfg.ServerAddr = globalCfg.ServerAddr
		cfg.DeviceID = globalCfg.DeviceID
		cfg.BinDir = globalCfg.BinDir

		if cfg.HomeDir == "" {
			cfg.HomeDir = filepath.Join(globalCfg.HomeDir, cfg.AppName)
		} else {
			cfg.HomeDir = filepath.Join(cfg.HomeDir, cfg.AppName)
		}

		zlog.Infof("config file %s applied", cfgFile)

		return cfg, nil
	}

	return nil, fmt.Errorf("no config files found from: %v", cfgFiles)
}

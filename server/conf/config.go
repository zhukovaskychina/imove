package conf

import (
	"errors"
	"fmt"
	"log"
	"gopkg.in/ini.v1"
	"net"
	"os"
	"path"
	"path/filepath"
)
var ConfigPath string

type CommandLineArgs struct {
	ConfigPath string
}

type Cfg struct {
	Raw            *ini.File
	Logger         log.Logger
	BindAddress    string
	Port		   string
	DataBaseHost   string
	DataBasePort   int
	DatabaseName string
	DataBaseUser   string
	DataBasePassword string
	DBFilePath	string
}
func NewCfg() *Cfg {

	return nil
}
func (cfg *Cfg) Load(args *CommandLineArgs) *Cfg {
	setHomePath(args)
	iniFile, err := cfg.loadConfiguration(args)
	if err != nil {
		fmt.Println("加载配置文件时有异常", err)
		os.Exit(1)
	}
	cfg.Raw = iniFile
	cfg = cfg.parseCfg(cfg.Raw.Section("mysqld"))
	return cfg

}

func (cfg *Cfg) parseCfg(section *ini.Section) *Cfg {
	bindAdress, err := valueAsString(section, "bind-address", "localhost")
	if err != nil {
		fmt.Println("读取地址异常", err)
		os.Exit(1)
	}
	ip := net.ParseIP(bindAdress)
	if ip == nil {
		fmt.Println("IP地址异常", err)
		os.Exit(1)
	}

	cfg.BindAddress = bindAdress




	return cfg
}

func (cfg *Cfg) loadConfiguration(args *CommandLineArgs) (*ini.File, error) {
	var err error

	defaultConfigFile := path.Join(args.ConfigPath, "my.ini")

	// check if config file exists
	if _, err := os.Stat(defaultConfigFile); os.IsNotExist(err) {
		fmt.Println("imove-server Init Failed: Could not find config defaults, make sure homepath command line parameter is set or working directory is homepath")
		os.Exit(1)
	}

	// load defaults
	parsedFile, err := ini.Load(defaultConfigFile)
	if err != nil {
		fmt.Println(fmt.Sprintf("Failed to parse defaults.ini, %v", err))
		os.Exit(1)
		return nil, err
	}
	return parsedFile, err
}

func setHomePath(args *CommandLineArgs) {
	if args.ConfigPath != "" {
		ConfigPath = args.ConfigPath
		return
	}

	ConfigPath, _ = filepath.Abs(".")

}
func valueAsString(section *ini.Section, keyName string, defaultValue string) (value string, err error) {
	defer func() {
		if err_ := recover(); err_ != nil {
			err = errors.New("Invalid value for key '" + keyName + "' in configuration file")
		}
	}()

	return section.Key(keyName).MustString(defaultValue), nil
}
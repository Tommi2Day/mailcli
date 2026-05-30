// Package cmd commands
package cmd

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/tommi2day/gomodules/common"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

const (
	configEnvPrefix = "MAILCLI"
	configName      = "mailcli"
	configType      = "yaml"
)

var (
	// RootCmd is exported for use in tests
	RootCmd = &cobra.Command{
		Use:   "mailcli",
		Short: "mailcli – CLI for sending and reading mails",
		Long:  `mailcli is a command-line tool for sending emails via SMTP and reading them via IMAP`,
	}

	debugFlag      = false
	infoFlag       = false
	cfgFile        = ""
	noLogColorFlag = false
	unitTestFlag   = false
)

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().BoolVarP(&debugFlag, "debug", "", false, "verbose debug output")
	RootCmd.PersistentFlags().BoolVarP(&infoFlag, "info", "", false, "reduced info output")
	RootCmd.PersistentFlags().BoolVarP(&unitTestFlag, "unit-test", "", false, "redirect output for unit tests")
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file")
	RootCmd.PersistentFlags().BoolVarP(&noLogColorFlag, "no-color", "", false, "disable colored log output")

	if err := viper.BindPFlags(RootCmd.PersistentFlags()); err != nil {
		log.Fatal(err)
	}
}

// Execute runs the application
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetConfigType(configType)
	viper.SetConfigName(configName)
	if cfgFile == "" {
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		viper.AddConfigPath(".")
		viper.AddConfigPath(path.Join(home, ".config"))
		viper.AddConfigPath(path.Join(home, "etc"))
		viper.AddConfigPath("/etc")
	} else {
		viper.SetConfigFile(cfgFile)
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix(configEnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	haveConfig, err := processConfig()
	processFlags()

	log.SetLevel(log.ErrorLevel)
	switch {
	case viper.GetBool("debug"):
		log.SetReportCaller(true)
		log.SetLevel(log.DebugLevel)
	case viper.GetBool("info"):
		log.SetLevel(log.InfoLevel)
	}
	logFormatter := &prefixed.TextFormatter{
		DisableColors:   noLogColorFlag,
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	}
	log.SetFormatter(logFormatter)
	if unitTestFlag {
		log.SetOutput(RootCmd.OutOrStdout())
	}

	if haveConfig {
		log.Debugf("found configfile %s", cfgFile)
	} else {
		log.Debugf("Error using %s config: %s", configType, err)
	}
}

func processConfig() (bool, error) {
	err := viper.ReadInConfig()
	haveConfig := false
	if err == nil {
		cfgFile = viper.ConfigFileUsed()
		haveConfig = true
		viper.Set("config", cfgFile)
	}
	return haveConfig, err
}

func processFlags() {
	if common.CmdFlagChanged(RootCmd, "debug") {
		viper.Set("debug", debugFlag)
	}
	if common.CmdFlagChanged(RootCmd, "info") {
		viper.Set("info", infoFlag)
	}
	if common.CmdFlagChanged(RootCmd, "no-color") {
		viper.Set("no-color", noLogColorFlag)
	}
	debugFlag = viper.GetBool("debug")
	infoFlag = viper.GetBool("info")
}

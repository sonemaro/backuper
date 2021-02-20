package main

import (
	"fmt"
	"os"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	verbose bool
)

func main() {
	setupLog()
	setupCLI()
}

func setupLog() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
}

func setupCLI() {
	cobra.OnInitialize(initConfig)

	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup db and website files",
		Long: `Start the backup process and create backup files from
		website files and database and transfer them to a remote 
		server via SCP.`,
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(args)
		},
	}

	var rootCmd = &cobra.Command{}
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.backuper/backuper.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	rootCmd.AddCommand(backupCmd)
	rootCmd.Execute()
}

func initConfig() {
	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)
		appName := "backuper"
		viper.AddConfigPath(fmt.Sprintf("%s/.%s/", home, appName))
		viper.SetConfigName("backuper.json")
		viper.SetConfigType("json")
		viper.SetEnvPrefix(appName)
		viper.AutomaticEnv()
		viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	}

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.WithFields(log.Fields{
				"confName": cfgFile,
			}).Error("cannot find config file")
			os.Exit(1)
		} else {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("viper error")
			os.Exit(1)
		}
	}
}

package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	homedir "github.com/mitchellh/go-homedir"
	log "github.com/sirupsen/logrus"
	"github.com/sonemaro/backuper"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// TMP is our temporary folder name
	TMP = "tmp"
	// BACKUPS our backups store in this folder
	BACKUPS = "backups"
	// SSHTimeout wiil be used if no value
	// provided in config
	SSHTimeout = 120 * time.Second
	// DefaultAppName will be used if no value
	// provided in config
	DefaultAppName = ".backuper"
)

var (
	cfgFile  string
	verbose  bool
	jsonLogs bool
)

// Config holds our config data
type Config struct {
	SitePath       string
	DBUsername     string
	DBPassword     string
	DBHostname     string
	DBName         string
	DBPort         int
	SSHUsername    string
	SSHKeyPath     string
	SSHRemote      string
	SSHTimeout     time.Duration
	SSHDestination string
	SSHInsecure    bool
	AppName        string
	AppHome        string
}

// Setup is responsible for setting up everything
// and provides utilities to work with backup files
// and things like that
type Setup struct {
	Config  Config
	AppHome string
}

func generateTimeName(prefix string, ext string) string {
	t := time.Now().Format("20060102T150405")
	return fmt.Sprintf("%s%s.%s", prefix, t, ext)
}

func (s *Setup) getAppHome() string {
	return path.Join(s.AppHome, s.Config.AppName)
}

func (s *Setup) getSubFolder(fld string) string {
	return path.Join(s.getAppHome(), fld)
}

// Backup starts the backup process.
func (s *Setup) Backup() error {
	arc := backuper.Archiver{
		// IOCopyProxy: backuper.IOCopyProgress,
	}

	// generate a time base name for our site backup
	// example: $HOME/.backuper/tmp/site_0210221T175418.zip
	siteBackup := path.Join(s.getSubFolder(TMP), generateTimeName("site_", "zip"))

	// create a zipfile containing all website files to our tmp folder
	err := arc.ZipDirectory(s.Config.SitePath, siteBackup)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("cannot create backup from website")
		return err
	}

	log.WithFields(log.Fields{
		"backupName": siteBackup,
	}).Info("website files have successfully added to our zip file")

	log.Debug("going to create db dump")
	dbdump := backuper.NewDBDump(s.Config.DBUsername,
		s.Config.DBPassword,
		s.Config.DBHostname,
		s.Config.DBName,
		s.Config.DBPort,
	)

	dumpName, err := dbdump.Dump(s.getSubFolder(TMP))
	if err != nil {
		log.WithFields(log.Fields{
			"error":    err,
			"dumpName": dumpName,
		}).Error("cannot create db dump")
		return err
	}
	log.WithFields(log.Fields{
		"dumpName": dumpName,
	}).Info("dbdump created successfully")

	log.Debug("going to create our final zip file")
	// example : $HOME/.backuper/backups/final_0210221T175418.zip
	finalDst := path.Join(s.getSubFolder(BACKUPS), generateTimeName("final_", "zip"))

	finalZipContents := []string{siteBackup, dumpName}
	fi, err := backuper.ZipFiles(finalDst, finalZipContents)
	if err != nil {
		log.WithFields(log.Fields{
			"error":       err,
			"files":       finalZipContents,
			"destination": finalDst,
		}).Error("cannot create final zip file")
		return err
	}

	log.WithFields(log.Fields{
		"size": fmt.Sprintf("%dMB", fi.Size()/1024),
		"name": fi.Name(),
	}).Info("final zip created successfully")

	log.Debug("going to transfer the final file to our remote server")

	scpu := backuper.SCPUtil{
		PrivateKey: s.Config.SSHKeyPath,
		Remote:     s.Config.SSHRemote,
		Username:   s.Config.SSHUsername,
		Timeout:    s.Config.SSHTimeout,
	}

	ff, err := os.Open(finalDst)
	defer ff.Close()

	if err != nil {
		log.WithFields(log.Fields{
			"error":     err,
			"finalFile": finalDst,
		}).Error("cannot open final file")
		return err
	}

	st, err := ff.Stat()
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"file":  ff,
		}).Error("cannot obtain stat")
		return err
	}

	bar := pb.Full.Start64(st.Size())
	barReader := bar.NewProxyReader(ff)

	// example: /home/ubuntu/final_20210222T163735.zip
	var remoteDst string

	// check if destination is not provided
	if s.Config.SSHDestination == "" {
		remoteDst = fmt.Sprintf("/home/%s/%s", s.Config.SSHUsername, fi.Name())
	} else {
		remoteDst = path.Join(s.Config.SSHDestination, fi.Name())
	}

	err = scpu.Copy(barReader, remoteDst, st.Size())
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
			"file":  finalDst,
		}).Error("cannot transfer final zip")
		// bar.Finish()
		return err
	}
	bar.Finish()

	log.Info("backup transferred successfully to our remote server")
	return nil
}

// CleanUp removes all files in tmp folder
func (s *Setup) CleanUp() error {
	err := removeContents([]string{s.getSubFolder(TMP), s.getSubFolder(BACKUPS)})
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("cleanup error")
		return err
	}
	log.Info("cleanup successful")
	return err
}

// SetupLog prepares our log
func (*Setup) SetupLog() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

// SetupCLI implements Cobra and sets flags and cli options
func (s *Setup) SetupCLI() {
	cobra.OnInitialize(s.initConfig)

	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup db and website files",
		Long: `Start the backup process and create backup files from
		website files and database and transfer them to a remote 
		server via SCP.`,
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			s.Backup()
		},
	}

	cleanupCmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Removes backups and temporary files",
		Long: `backuper does not touch final and temporary files
		after each backup so this command is useful if you want
		to remove these files`,
		Args: cobra.MinimumNArgs(0),
		Run: func(cmd *cobra.Command, args []string) {
			s.CleanUp()
		},
	}

	var rootCmd = &cobra.Command{}
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.backuper/backuper.json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVarP(&jsonLogs, "json", "j", false, "json output")

	rootCmd.AddCommand(backupCmd, cleanupCmd)
	rootCmd.Execute()
}

// initConfig will be executed on Cobra initiation
func (s *Setup) initConfig() {
	defaultAppName := ".backuper"
	confName := "backuper.json"
	home, err := homedir.Dir()
	cobra.CheckErr(err)

	if verbose {
		log.SetLevel(log.DebugLevel)
	}
	if jsonLogs {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(path.Join(home, defaultAppName))
		viper.SetConfigName(confName)
	}

	var paths []string
	// example: /home/sonemaro/.backuper
	appHomeFull := path.Join(home, defaultAppName)
	tmp := path.Join(appHomeFull, TMP)
	backups := path.Join(appHomeFull, BACKUPS)

	// Set AppHome
	s.AppHome = appHomeFull

	if !isExist(appHomeFull) {
		log.WithFields(log.Fields{
			"home":         appHomeFull,
			"backuperHome": appHomeFull,
			"tmp":          tmp,
			"backups":      backups,
		}).Debug("going to create necessary folders")

		paths = []string{appHomeFull, tmp, backups}
		for _, p := range paths {
			err := mkdirIfNotExist(p)
			if err != nil {
				log.WithFields(log.Fields{
					"folders": paths,
				}).Error("cannot create necessary folders")
				os.Exit(1)
			}
		}
	}

	viper.SetDefault("appName", DefaultAppName)
	viper.SetDefault("ssh.timeout", SSHTimeout)
	viper.SetConfigType("json")
	viper.SetEnvPrefix(defaultAppName)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.WithFields(log.Fields{
				"confName": confName,
			}).Error("cannot find config file")
			os.Exit(1)
		} else {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("viper error")
			os.Exit(1)
		}
	}

	s.Config = Config{
		SitePath:       viper.GetString("site"),
		DBUsername:     viper.GetString("db.username"),
		DBPassword:     viper.GetString("db.password"),
		DBHostname:     viper.GetString("db.hostname"),
		DBName:         viper.GetString("db.dbname"),
		DBPort:         viper.GetInt("db.port"),
		SSHUsername:    viper.GetString("ssh.username"),
		SSHKeyPath:     viper.GetString("ssh.key"),
		SSHRemote:      viper.GetString("ssh.remote"),
		SSHTimeout:     viper.GetDuration("ssh.timeout") * time.Second,
		SSHInsecure:    viper.GetBool("ssh.insecure"),
		SSHDestination: viper.GetString("ssh.destination"),
		AppHome:        viper.GetString("app.home"),
		AppName:        viper.GetString("app.name"),
	}
}

func mkdirIfNotExist(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("cannot create .backuper folder")
			return err
		}
	}
	return nil
}

func isExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

func removeContents(dirs []string) error {
	for _, dir := range dirs {
		d, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer d.Close()
		names, err := d.Readdirnames(-1)
		if err != nil {
			return err
		}
		for _, name := range names {
			err = os.RemoveAll(filepath.Join(dir, name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

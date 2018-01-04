package main

import (
	"fmt"
	"github.com/multiplay/go-slack/chat"
	"github.com/multiplay/go-slack/lrhook"
	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

type FileInfo struct {
	Name     string
	FullName string
	Size     int64
	Mode     string
	ModTime  time.Time
	IsDir    bool
	Age      uint64 //age in seconds
}

func visit(files *[]FileInfo) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		var result = FileInfo{
			Name:     info.Name(),
			FullName: path,
			Size:     info.Size(),
			Mode:     info.Mode().String(),
			ModTime:  info.ModTime(),
			Age:      uint64(time.Now().Sub(info.ModTime()).Seconds()),
			IsDir:    info.IsDir(),
		}
		*files = append(*files, result)
		return nil
	}
}

func main() {
	var (
		poll           uint64
		maxAge         uint64
		logLevel       string
		ageWarning     string
		slackWarnLevel string
		slackWebhook   string
		slackChannel   string
		slackIcon      string
		folders        []string
		folder         string
		ignoreFolders  bool
	)

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Printf("This tool is used to poll the given directories for files that are older than the given max age.\n")
		fmt.Printf("The output log level and the notifications via slack can be configured and all parameters\n")
		fmt.Printf("can be set via environment variables with the same name in capitals as well as in a \n")
		fmt.Printf("config file %s.conf with name space value format.\n", os.Args[0])
		fmt.Printf("    %s --poll=60 --maxAge=1800 directory1 directory2 ...\n", os.Args[0])
		fmt.Printf("    POLL=60 MAXAGE=1800 DIRECTORY=/watch %s directory1 directory2 ...\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Uint64Var(&poll, "poll", 60, "Specify polling interval in seconds, defaults to 60")
	flag.Uint64Var(&maxAge, "maxAge", 1800, "Specify max age of a file in seconds, defaults to 1800 (30min)")
	flag.StringVar(&logLevel, "logLevel", "info", "Specify the log levels. Possible values are 'panic', 'fatal', 'error', 'warn', 'info', 'debug'. Defaults to 'info'")
	flag.StringVar(&ageWarning, "ageWarning", "File age", "Specify the warning display if file is too old")
	flag.StringVar(&slackWarnLevel, "slackWarnLevel", "error", "Specify the log level from which on slack messages should be sent. Possible values are 'panic', 'fatal', 'error', 'warn', 'info', 'debug'. Defaults to 'info'")
	flag.StringVar(&slackWebhook, "slackWebhook", "", "Specify the slack webhook.")
	flag.StringVar(&slackChannel, "slackChannel", "alerts", "Specify the slack channel to post to")
	flag.StringVar(&slackIcon, "slackIcon", ":ghost:", "Specify the slack message icon")
	flag.StringVar(&folder, "directory", "", "Directory to be polled. Is added to the args list.")
	flag.BoolVar(&ignoreFolders, "ignoreFolders", true, "Ignore directories and just check files for age, defaults to true")
	flag.Parse()
	loglevel, err := log.ParseLevel(logLevel)
	if err != nil {
		log.WithFields(log.Fields{
			"logLevel": logLevel,
		}).Panic("Invalid log level!")
		flag.Usage()
		os.Exit(99)
	}
	log.SetLevel(loglevel)
	log.SetFormatter(&log.JSONFormatter{})
	if slackWebhook != "" {
		slacklevel, err := log.ParseLevel(slackWarnLevel)
		if err != nil {
			log.WithFields(log.Fields{
				"slackWarnLevel": slackWarnLevel,
			}).Panic("Invalid log level!")
			flag.Usage()
			os.Exit(99)
		}
		cfg := lrhook.Config{
			MinLevel: slacklevel,
			Message: chat.Message{
				Channel:   "#" + slackChannel,
				IconEmoji: slackIcon,
			},
		}
		h := lrhook.New(cfg, slackWebhook)
		log.AddHook(h)
	}

	if flag.NArg() == 0 && folder == "" {
		log.Panic("No directory given!")
		flag.Usage()
		os.Exit(1)
	}
	folders = flag.Args()
	if folder != "" {
		folders = append(folders, folder)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("\nreceived %s, exiting...\n", sig)
		os.Exit(1)
	}()

	fmt.Println("Watching the following directories:")
	for _, path := range folders {
		fmt.Printf("- %s \n", path)
	}
	fmt.Printf("Press Ctrl+C to end\n")

	for {
		var files []FileInfo
		for _, path := range folders {
			err := filepath.Walk(path, visit(&files))
			if err != nil {
				log.WithFields(log.Fields{
					"path": path,
					"err":  err,
				}).Panic("Unable to walk through directory!")
			}
		}
		for _, file := range files {
			log.WithFields(log.Fields{
				"filename": file.FullName,
				"mode":     file.Mode,
				"ModTime":  file.ModTime,
			}).Info("")
			if file.Age > maxAge {
				if !(file.IsDir && ignoreFolders) {
					log.WithFields(log.Fields{
						"filename": file.FullName,
						"mode":     file.Mode,
						"ModTime":  file.ModTime,
					}).Error(ageWarning)
				}
			}
		}
		time.Sleep(time.Duration(poll) * time.Second)
	}
}

package main

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/jedib0t/go-pretty/v6/table"
	log "github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
	"github.com/urfave/cli/v2"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var Version = ""
var GitCommit = ""

func GetVersion() string {
	return fmt.Sprintf("%s,%s", Version, GitCommit)
}

var config Config

type GlobalOptions struct {
	ShowAll        bool
	GroupFilter    []string
	DatasetFilter  []string
	FilenameFilter []string
	Since          *time.Time
}

func main() {
	log.SetLevel(log.InfoLevel)
	formatter := &easy.Formatter{
		LogFormat: "%msg%\n",
	}
	log.SetFormatter(formatter)

	cliApp := &cli.App{
		Name:    "speedtest-extract",
		Usage:   "Download extract files for Speedtest Intelligence",
		Version: GetVersion(),
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config",
				Usage:    "Specify the config file",
				Required: false,
				Value:    DefaultConfigFile,
			},
			&cli.BoolFlag{
				Name:  "all",
				Usage: "Show all extract files, not just latest available",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "filter-groups",
				Usage: "Limit extracts to this comma-delimited list of groups",
			},
			&cli.StringFlag{
				Name:  "filter-datasets",
				Usage: "Limit extracts to this comma-delimited list of datasets",
			},
			&cli.StringFlag{
				Name:  "filter-filenames",
				Usage: "Limit extracts to this comma-delimited list of filenames",
			},
			&cli.StringFlag{
				Name:  "since",
				Usage: "Limit extracts to ones updates since the provided date (YYYY-MM-DD)",
			},
			&cli.BoolFlag{
				Name:  "verbose",
				Usage: "Enable verbose logging to help with debugging",
				Value: false,
			},
		},
		Before: func(context *cli.Context) error {
			configFile := context.String("config")
			c, err := ReadConfig(configFile)
			if err != nil {
				if os.IsNotExist(err) {
					err := WriteConfig()
					if err != nil {
						return err
					}
					return fmt.Errorf("config file not found, wrote default values to %s", configFile)
				}
				return err
			}
			config = *c

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:   "list",
				Action: ListExtracts,
				Usage:  "List available extracts",
			},
			{
				Name:   "download",
				Action: DownloadExtracts,
				Usage:  "Download extract files",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "overwrite-existing",
						Usage: "Re-download existing extract files with the same name",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "confirm",
						Usage: "Don't prompt to confirm downloads",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "use-file-hierarchy",
						Usage: "Download files into a hierarchy based on the group and dataset names vs a flat list",
						Value: false,
					},
					&cli.IntFlag{
						Name:  "concurrency",
						Usage: "Set the number of concurrent downloads",
						Value: 1,
					},
				},
			},
		},
	}
	err := cliApp.Run(os.Args)
	if err != nil {
		log.WithError(err).Error(err)
	}
}

func RedirectLoggingPolicy() resty.RedirectPolicy {
	return resty.RedirectPolicyFunc(func(req *http.Request, via []*http.Request) error {
		if len(via) > 0 {
			log.Debug(fmt.Sprintf("redirecting from %s to %s", via[len(via)-1].URL, req.URL))
		}
		return nil
	})
}

func GetClient(downloadClient bool) *resty.Client {
	client := resty.New()
	if !downloadClient { //we only need to send these headers to the extract service, not for file download
		client.SetBasicAuth(config.ApiKey, config.ApiSecret)
		client.SetHeader("Content-Type", "application/json")
	}
	client.SetHeader("User-Agent", fmt.Sprintf("ookla/speedtest-extract/%s", GetVersion()))
	client.SetRedirectPolicy(RedirectLoggingPolicy())

	return client
}

func GetGlobalOptions(context *cli.Context) (*GlobalOptions, error) {
	showAll := context.Bool("all")
	groupFilter := context.String("filter-groups")
	datasetFilter := context.String("filter-datasets")
	filenameFilter := context.String("filter-filenames")
	since := context.String("since")
	verbose := context.Bool("verbose")

	args := &GlobalOptions{
		ShowAll: showAll,
	}
	if len(groupFilter) > 0 {
		args.GroupFilter = strings.Split(groupFilter, ",")
	}
	if len(datasetFilter) > 0 {
		args.DatasetFilter = strings.Split(datasetFilter, ",")
	}
	if len(filenameFilter) > 0 {
		args.FilenameFilter = strings.Split(filenameFilter, ",")
		//account for possibility that the user ignored the .zip extension for the filenames, so allow either way
		for _, f := range args.FilenameFilter {
			args.FilenameFilter = append(args.FilenameFilter, fmt.Sprintf("%s.zip", f))
		}
	}
	if len(since) > 0 {
		t, err := time.Parse("2006-01-02", since)
		if err != nil {
			return nil, err
		}
		args.Since = &t
	}

	if verbose {
		log.SetLevel(log.DebugLevel)
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp:          true,
			DisableLevelTruncation: true,
			PadLevelText:           true,
		})
	}

	log.WithFields(log.Fields{
		"all":            showAll,
		"groupFilter":    args.GroupFilter,
		"datasetFilter":  args.DatasetFilter,
		"filenameFilter": args.FilenameFilter,
		"since":          args.Since,
	}).Debug("global flags")

	return args, nil
}

func ListExtracts(context *cli.Context) error {
	return ExtractHandler(context, "list")
}

func DownloadExtracts(context *cli.Context) error {
	return ExtractHandler(context, "download")
}

func ListFiles(files []ExtractFile) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Groups", "Dataset", "File", "Updated", "Latest"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true},
		{Number: 2, AutoMerge: true},
	})
	for _, f := range files {
		latest := ""
		if f.Latest {
			latest = "*"
		}
		groups := strings.Join(f.Item.Groups, ", ")
		row := table.Row{
			groups, f.Dataset, f.Name, f.Updated, latest,
		}
		t.AppendRow(row)
	}
	t.Render()
}

func downloadWorker(workChan <-chan ExtractFile, resultChan chan<- DownloadResult, downloadClient *resty.Client, useFileHierarchy bool, overwriteExisting bool) {
	for w := range workChan {
		result := w.Download(downloadClient, useFileHierarchy, overwriteExisting)
		if result.err != nil {
			log.WithError(result.err).Error(fmt.Sprintf("error downloading %s", w.Item.Name))
		}
		resultChan <- result
	}
}

func ExtractHandler(context *cli.Context, command string) error {
	args, err := GetGlobalOptions(context)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"extractUrl":           config.ExtractUrl,
		"storageDirectory":     config.StorageDirectory,
		"cacheDurationMinutes": config.CacheDurationMinutes,
		"cacheFilename":        config.CacheFilename,
	}).Debug("config values")

	cache := ReadExtractsCache()
	client := GetClient(false)
	log.Debug(fmt.Sprintf("Client headers: %s", client.Header))
	extracts, err := GetExtracts(client, "", cache)
	if err != nil {
		return err
	}
	WriteExtractsCache(cache)
	files := FilterFiles(extracts, args.GroupFilter, args.DatasetFilter, args.FilenameFilter, args.Since, !args.ShowAll, nil)

	if len(files) == 0 {
		return ErrNoMatchingFiles
	}

	if command == "list" {
		ListFiles(files)
	} else if command == "download" {
		log.Info(fmt.Sprintf("Found %d file(s)", len(files)))
		overwriteExisting := context.Bool("overwrite-existing")
		confirm := context.Bool("confirm")
		useFileHierarchy := context.Bool("use-file-hierarchy")
		concurrency := context.Int("concurrency")

		log.WithFields(log.Fields{
			"overwriteExisting": overwriteExisting,
			"confirm":           confirm,
			"useFileHierarchy":  useFileHierarchy,
			"concurrency":       concurrency,
		}).Debug("download flags")

		download := false
		if confirm {
			download = true
		} else {
			prompt := "\nProceed with download? [(y)es|(n)o|(l)ist]: "
			resp := GetInput(prompt, []string{"yes", "no", "list"}, true)
			if resp == "y" {
				download = true
			} else if resp == "l" {
				ListFiles(files)
				prompt = "\nProceed with download? [(y)es|(n)o]"
				resp = GetInput(prompt, []string{"yes", "no"}, true)
				if resp == "y" {
					download = true
				}
			}
		}

		if download {
			downloadClient := GetClient(true)
			log.Debug(fmt.Sprintf("Download client headers: %s", downloadClient.Header))

			downloadChan := make(chan ExtractFile, len(files))
			resultChan := make(chan DownloadResult, len(files))
			var wg sync.WaitGroup
			for i := range concurrency {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					downloadWorker(downloadChan, resultChan, downloadClient, useFileHierarchy, overwriteExisting)
				}(i)
			}

			for _, f := range files {
				log.Debug(fmt.Sprintf("Adding %s to download queue", f.Name))
				downloadChan <- f
			}
			close(downloadChan)
			wg.Wait()
			close(resultChan)

			downloaded := 0
			skipped := 0
			errors := 0
			for result := range resultChan {
				success, err := result.success, result.err
				if err != nil {
					errors += 1
				} else {
					if success {
						downloaded += 1
					} else {
						skipped += 1
					}
				}
			}
			log.Info(fmt.Sprintf("Downloaded %d file(s), skipped %d existing file(s), encountered %d error(s)", downloaded, skipped, errors))
		}
	}

	return nil
}

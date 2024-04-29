package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ExtractItem struct {
	Name     string                    `json:"name"`
	Url      string                    `json:"url"`
	Type     string                    `json:"type"`
	Modified int64                     `json:"mtime"`
	Size     int64                     `json:"size"`
	Datasets map[string][]*ExtractItem `json:"-"`
	Latest   map[string]*ExtractItem   `json:"-"`
	Children []*ExtractItem            `json:"-"`
	Groups   []string                  `json:"-"`
}

type ExtractFile struct {
	Dataset string
	Name    string
	Latest  bool
	Updated time.Time
	Item    *ExtractItem
}

type ExtractsCache struct {
	Timestamp time.Time                 `json:"timestamp"`
	Responses map[string][]*ExtractItem `json:"responses"`
}

func (e *ExtractItem) IsDirectory() bool {
	return e.Type == "dir"
}

func (e *ExtractItem) IsDataset() bool {
	return e.Type == "file" &&
		!strings.Contains(e.Name, "headers") && //ignore header files
		strings.Contains(e.Name, "_20") //ensure it's a filename containing a date to filter non-relevant files
}

func (e *ExtractItem) DatasetName() string {
	return e.Name[:strings.Index(e.Name, "_20")]
}

func (e *ExtractFile) Download(client *resty.Client, useFileHierarchy bool, overwriteExisting bool) error {
	item := e.Item
	if item.IsDataset() {
		paths := []string{config.StorageDirectory}
		if useFileHierarchy {
			paths = append(paths, e.Item.Groups...)
			paths = append(paths, e.Dataset)
		}
		var path string
		//did not use MkDirAll due to issues w/ umask filtering and dealing with diff platforms (windows)
		for _, p := range paths {
			path = filepath.Join(path, p)
			err := os.Mkdir(path, 0700)
			errors.Is(err, os.ErrExist)
			if err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
		}

		fileName := filepath.Join(path, e.Name)
		_, err := os.Stat(fileName)
		if overwriteExisting || (err != nil && errors.Is(err, os.ErrNotExist)) {
			log.Info(fmt.Sprintf("Downloading %s to %s", e.Name, path))
			_, err = client.R().
				SetOutput(fileName).
				Get(item.Url)
			if err != nil {
				return err
			}
			log.Info(fmt.Sprintf("%s complete", e.Name))
			stats, err := os.Stat(fileName)
			if err != nil {
				return err
			}
			downloadSize := stats.Size()
			if item.Size != downloadSize {
				return fmt.Errorf("filesize mismatch for %s. expected: %d, received: %d", e.Name, item.Size, downloadSize)
			}
		} else {
			log.Info(fmt.Sprintf("%s exists, skipping. re-run with --overwrite-existing to download anyway", e.Name))
		}
	}
	return nil
}

func WriteExtractsCache(cache *ExtractsCache) {
	if config.CacheDurationMinutes > 0 {
		cacheFilename := config.CacheFilename
		if cache != nil && cache.Timestamp.IsZero() {
			cache.Timestamp = time.Now().UTC()
			out, err := json.Marshal(cache)
			if err == nil {
				_ = os.WriteFile(cacheFilename, out, 0644)
			}
		}
	}
}

func ReadExtractsCache() *ExtractsCache {
	var cache *ExtractsCache
	if config.CacheDurationMinutes > 0 {
		log.Debug("cache enabled")
		hasCache := false
		cacheFilename := config.CacheFilename
		cacheFile, err := os.ReadFile(cacheFilename)
		if err == nil {
			log.Debug(fmt.Sprintf("found cache file %s", cacheFilename))
			err = json.Unmarshal(cacheFile, &cache)
			if err == nil {
				log.Debug("parsed cache file")
				now := time.Now().UTC()
				cacheDiff := now.Sub(cache.Timestamp)
				if cacheDiff < time.Duration(config.CacheDurationMinutes)*time.Minute {
					log.Info(fmt.Sprintf("Using cached request from %s", cache.Timestamp))
					hasCache = true
				} else {
					log.Debug("cache file too old, ignoring")
				}
			}
		}
		if !hasCache { //caching is enabled, but we did not find a valid cache, start a new one
			log.Debug("no valid cache file found, creating new")
			cache = &ExtractsCache{
				Timestamp: time.Time{},
				Responses: make(map[string][]*ExtractItem),
			}
		}
	}
	return cache
}

func GetExtracts(client *resty.Client, path string, cache *ExtractsCache) ([]*ExtractItem, error) {
	var extracts []*ExtractItem
	url := config.ExtractUrl + path

	cacheValid := false
	if cache != nil { //caching is enabled
		if !cache.Timestamp.IsZero() { //we found an existing cache
			if extractItems, ok := cache.Responses[url]; ok {
				log.Debug(fmt.Sprintf("using cached data from %s", url))
				extracts = extractItems
				cacheValid = true
			}
		}
	}

	if !cacheValid {
		log.Debug(fmt.Sprintf("requesting data from %s", url))
		resp, err := client.R().
			SetResult(&extracts).
			Get(url)
		if err != nil {
			log.WithError(err).Debug(fmt.Sprintf("error retrieving extract data from %s", url))
			return nil, err
		}
		if len(path) == 0 && resp.IsError() {
			switch resp.StatusCode() {
			case 401, 403:
				err = ErrAuth
			case 404:
				err = ErrNoExtract
			case 500:
				err = ErrServerError
			default:
				err = ErrUnknownStatus
			}
			log.WithError(err).Debug(fmt.Sprintf("error retrieving extract data from %s", url))
			return nil, err
		}

		if cache != nil { //cache enabled, but we either had no existing cache or it was invalid, start a new one
			cache.Timestamp = time.Time{}
			cache.Responses[url] = extracts
		}
	}
	log.Debug(fmt.Sprintf("found %d items in index", len(extracts)))

	for i, e := range extracts {
		if e.IsDirectory() {
			if len(e.Groups) == 0 {
				e.Groups = strings.Split(strings.Trim(e.Url, "/"), "/")
			}

			subDir := e.Url
			children, err := GetExtracts(client, subDir, cache)
			if err != nil {
				return nil, err
			}
			extracts[i].Children = make([]*ExtractItem, 0)
			for _, child := range children {
				if child.IsDirectory() || child.IsDataset() {
					extracts[i].Children = append(extracts[i].Children, child)

					groupUrl := child.Url
					if child.IsDataset() {
						groupUrl = subDir
					}
					trimmed := strings.Trim(groupUrl, "/")
					child.Groups = strings.Split(trimmed, "/")
				}

				if child.IsDataset() {
					name := child.DatasetName()
					if extracts[i].Latest == nil {
						extracts[i].Latest = make(map[string]*ExtractItem, 0)
						extracts[i].Datasets = make(map[string][]*ExtractItem, 0)
					}
					extracts[i].Datasets[name] = append(extracts[i].Datasets[name], child)

					if latest, ok := extracts[i].Latest[name]; !ok || extracts[i].Modified > latest.Modified {
						extracts[i].Latest[name] = child
					}
				}
			}
		}
	}
	return extracts, nil
}

func FilterFiles(items []*ExtractItem, groupFilter []string, filenameFilter []string, since *time.Time, latestOnly bool, files []ExtractFile) []ExtractFile {
	if since != nil {
		latestOnly = false
	}
	if files == nil {
		files = make([]ExtractFile, 0)
	}
	for _, i := range items {
		if i.IsDirectory() {
			groups := i.Groups

			matchedGroup := false
			if len(groupFilter) == 0 {
				matchedGroup = true
			}
			//TODO this group filter matches any of the terms, is there a need for an "all" filter?
			for _, g := range groups {
				if contains(g, groupFilter) {
					matchedGroup = true
				}
			}

			if matchedGroup {
				for name, datasets := range i.Datasets {
					dataset := name
					for _, d := range datasets {
						filename := d.Name
						if len(filenameFilter) == 0 || contains(filename, filenameFilter) {
							if !latestOnly || d == i.Latest[name] {
								updated := time.UnixMilli(d.Modified).UTC()
								if since == nil || updated.Equal(*since) || updated.After(*since) {
									log.WithFields(log.Fields{
										"groups":  groups,
										"dataset": dataset,
										"latest":  d == i.Latest[name],
										"updated": updated,
									}).Debug(fmt.Sprintf("found file matching all filters: %s", filename))
									files = append(files, ExtractFile{
										Dataset: dataset,
										Name:    filename,
										Latest:  d == i.Latest[name],
										Updated: updated,
										Item:    d,
									})
								}
							}
						}
					}
				}
			}
			files = FilterFiles(i.Children, groupFilter, filenameFilter, since, latestOnly, files)
		}
	}
	return files
}

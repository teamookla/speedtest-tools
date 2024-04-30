package main

import (
	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type FilterCounts struct {
	Files    int
	Groups   int
	Datasets int
	Latest   int
}

func ReadResponseFixture(name string) ([]byte, error) {
	file := name + ".json"
	content, err := os.ReadFile(filepath.Join("fixtures", file))
	if err != nil {
		return nil, err
	}
	return content, nil
}

var MockServer = httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
	path := req.URL.Path

	fixture := strings.ReplaceAll(strings.Split(path, "/extracts")[1], "/", "")
	if len(fixture) == 0 {
		fixture = "extracts"
	}
	res.Header().Set("Content-Type", "application/json")

	contents, err := ReadResponseFixture(fixture)
	if err != nil {
		res.WriteHeader(404)
	} else {
		res.WriteHeader(200)
		_, _ = res.Write(contents)
	}
}))

func GetTestExtracts() ([]*ExtractItem, error) {
	config.ExtractUrl = MockServer.URL + "/extracts"
	config.CacheDurationMinutes = -1
	client := resty.New()
	client.SetHeader("Content-Type", "application/json")
	return GetExtracts(client, "", nil)
}

func RunFilters(t *testing.T, extracts []*ExtractItem, groupFilter []string, datasetFilter []string, filenameFilter []string, since *time.Time, latestOnly bool, expected FilterCounts) []ExtractFile {
	files := FilterFiles(extracts, groupFilter, datasetFilter, filenameFilter, since, latestOnly, nil)

	latestCount := 0
	groups := make(map[string]interface{})
	datasets := make(map[string]interface{})
	for _, f := range files {
		if f.Latest {
			latestCount += 1
		}
		//TODO this test doesn't properly account for more nested directories (multiple groups)
		for _, g := range f.Item.Groups {
			groups[g] = nil
		}
		datasets[f.Dataset] = nil
	}

	actual := FilterCounts{
		Files:    len(files),
		Groups:   len(groups),
		Datasets: len(datasets),
		Latest:   latestCount,
	}
	assert.Equal(t, expected.Files, actual.Files, "file count not equal")
	assert.Equal(t, expected.Groups, actual.Groups, "group count not equal")
	assert.Equal(t, expected.Datasets, actual.Datasets, "dataset count not equal")
	assert.Equal(t, expected.Latest, actual.Latest, "latest count not equal")

	return files
}

func TestGetExtracts(t *testing.T) {
	t.Run("should properly marshal extracts from json response", func(t *testing.T) {
		extracts, err := GetTestExtracts()
		assert.Nil(t, err)
		assert.Len(t, extracts, 4)
	})
}

func TestFilters(t *testing.T) {
	extracts, _ := GetTestExtracts()

	t.Run("should return all files when no filters are specified", func(t *testing.T) {
		_ = RunFilters(t, extracts, []string{}, []string{}, []string{}, nil, false, FilterCounts{
			Files:    24,
			Groups:   4,
			Datasets: 5,
			Latest:   5,
		})
	})

	t.Run("should filter for the latest files", func(t *testing.T) {
		files := RunFilters(t, extracts, []string{}, []string{}, []string{}, nil, true, FilterCounts{
			Files:    5,
			Groups:   4,
			Datasets: 5,
			Latest:   5,
		})
		for _, f := range files {
			assert.True(t, f.Latest)
		}
	})

	t.Run("should filter for specific groups", func(t *testing.T) {
		groups := []string{"android", "web"}
		files := RunFilters(t, extracts, groups, []string{}, []string{}, nil, false, FilterCounts{
			Files:    12,
			Groups:   2,
			Datasets: 2,
			Latest:   2,
		})
		for _, f := range files {
			matchedGroup := false
			for _, g := range f.Item.Groups {
				if contains(g, groups) {
					matchedGroup = true
				}
			}
			assert.True(t, matchedGroup, "all but android and web groups should be filtered")
		}
	})

	t.Run("should filter for specific datasets", func(t *testing.T) {
		datasets := []string{"desktop"}
		files := RunFilters(t, extracts, []string{}, datasets, []string{}, nil, false, FilterCounts{
			Files:    5,
			Groups:   1,
			Datasets: 1,
			Latest:   1,
		})
		for _, f := range files {
			assert.Contains(t, datasets, f.Dataset, "all but desktop datasets should be filtered")
		}
	})

	t.Run("should filter for files modified after a given date", func(t *testing.T) {
		since, _ := time.Parse("2006-01-02", "2022-05-01")
		files := RunFilters(t, extracts, []string{}, []string{}, []string{}, &since, false, FilterCounts{
			Files:    16,
			Groups:   4,
			Datasets: 5,
			Latest:   5,
		})
		for _, f := range files {
			assert.True(t, f.Updated.After(since), "all files modified before the since date should be filtered")
		}
	})

	t.Run("should filter for latest version of specific filenames", func(t *testing.T) {
		filenames := []string{"android_2022-05-01.zip", "desktop_2022-04-01.zip"}
		files := RunFilters(t, extracts, []string{}, []string{}, filenames, nil, true, FilterCounts{
			Files:    2,
			Groups:   2,
			Datasets: 2,
			Latest:   2,
		})
		for _, f := range files {
			assert.Contains(t, filenames, f.Name, "all but specific android and desktop file should be filtered")
		}
	})
}

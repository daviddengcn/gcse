package godocorg

import (
	"encoding/json"
	"net/http"

	"github.com/daviddengcn/gddo/doc"
	"github.com/golangplus/errors"
)

const (
	godocApiUrl = "http://api.godoc.org/packages"
)

// FetchAllPackagesInGodoc fetches the list of all packages on godoc.org
func FetchAllPackagesInGodoc(httpClient doc.HttpClient) ([]string, error) {
	req, err := http.NewRequest("GET", godocApiUrl, nil)
	if err != nil {
		return nil, errorsp.WithStacksAndMessage(err, "new request for %v failed", godocApiUrl)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errorsp.WithStacksAndMessage(err, "fetching %v failed", godocApiUrl)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, errorsp.NewWithStacks("StatusCode: %d", resp.StatusCode)
	}
	var results struct {
		Results []struct {
			Path string
		}
	}
	dec := json.NewDecoder(resp.Body)

	if err := dec.Decode(&results); err != nil {
		return nil, errorsp.WithStacks(err)
	}
	list := make([]string, 0, len(results.Results))
	for _, res := range results.Results {
		list = append(list, res.Path)
	}
	return list, nil
}

package util

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/hashicorp/go-version"
)

type githubResponse struct {
	Name string `json:"name"`
}

// CheckForRigUpdate checks to see if an upgrdate to rig is available, if so, return a message
func CheckForRigUpdate(curRigVersion string) string {
	// Local dev, version == "master" which isn't going to parse.
	curVer, verr := version.NewVersion(curRigVersion)
	if tag, err := currentRigReleaseTag(); err == nil {
		if tagVer, verr2 := version.NewVersion(tag); verr2 == nil {
			if verr != nil || tagVer.Compare(curVer) > 0 {
				return "An update for rig is available: " + tag
			}
		}
	} else {
		if Logger().IsVerbose {
			Logger().Warning("Can't parse released tag version: " + err.Error())
		}
	}
	return ""
}

// Return the current release tag for rig
func currentRigReleaseTag() (string, error) {
	// Fetch some json from github containing the latest release name
	url := "https://api.github.com/repos/phase2/rig/releases/latest"
	response, err := getRigReleaseTagResponse(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	// Collect the response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		if Logger().IsVerbose {
			Logger().Warning("ReadAll %s failed:\n%s", url, err)
		}
		return "", err
	}
	if response.StatusCode != 200 {
		if Logger().IsVerbose {
			Logger().Warning("ReadAll %s failed: %s", url, response.Status)
		}
		return "", errors.New(response.Status)
	}
	// Decode the json, pick off the name field
	decoder := githubResponse{}
	if err = json.Unmarshal(body, &decoder); err != nil {
		if Logger().IsVerbose {
			Logger().Warning("Unmarshal %s failed:\n%s", url, err)
		}
		return "", err
	}
	if Logger().IsVerbose {
		Logger().Info("rig current release tag: %s", decoder.Name)
	}
	return decoder.Name, nil
}

func getRigReleaseTagResponse(url string) (*http.Response, error) {
	client := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		if Logger().IsVerbose {
			Logger().Warning("NewRequest %s failed:\n%s", url, err)
		}
		return nil, err
	}
	// Execute the request
	response, err := client.Do(req)
	if err != nil {
		if Logger().IsVerbose {
			Logger().Warning("GET %s failed:\n%s", url, err)
		}
		return nil, err
	}
	return response, nil
}

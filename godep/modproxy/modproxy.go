package modproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Module struct {
	Path     string
	Versions []string
} //todo: add GetModule() with all information

type ModProxy struct {
	Endpoint string
}

//ListVersions of a modulePath as an unsorted []string, []string is nil when there are no versions
func (p ModProxy) ListVersions(ctx context.Context, modulePath string) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.Endpoint+"/"+modulePath+"/@v/list", nil)
	if err != nil {
		return nil, err
	}

	var res *http.Response
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err == nil {
			return
		}
		err = res.Body.Close()
	}()

	var resBytes []byte
	resBytes, err = io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if len(resBytes) <= 1 {
		err = res.Body.Close()
		if err != nil {
			return nil, err
		}
		return nil, nil
	}

	vs := strings.Split(string(resBytes[:len(resBytes)-1]), "\n")
	versions := make([]string, len(vs))
	copy(versions, vs)

	err = res.Body.Close()
	if err != nil {
		return nil, err
	}

	return versions, nil
}

type Info struct {
	//Version of Module
	Version string `json:"version"`
	//Time module was committed
	Time time.Time `json:"time"`
}

//GetLatestVersion of module defined by modulePath
func (p ModProxy) GetLatestVersion(ctx context.Context, modulePath string) (Info, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.Endpoint+"/"+modulePath+"/@latest", nil)
	if err != nil {
		return Info{}, err
	}

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return Info{}, err
	}

	defer func() {
		if err == nil {
			return
		}
		err = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return Info{}, fmt.Errorf("modproxy: module not found")
		}
		return Info{}, fmt.Errorf("modproxy: http server error")
	}

	decoder := json.NewDecoder(resp.Body)

	var info Info
	err = decoder.Decode(&info)
	if err != nil {
		return Info{}, err
	}

	err = resp.Body.Close()
	if err != nil {
		return Info{}, err
	}

	return info, nil
}

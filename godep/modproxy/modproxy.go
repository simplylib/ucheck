package modproxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/simplylib/multierror"
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
		return nil, fmt.Errorf("could not create new Request (%w)", err)
	}

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not Do HTTP Request (%W)", err)
	}

	defer func() {
		if err2 := resp.Body.Close(); err != nil {
			err = multierror.Append(err, fmt.Errorf("could not close Response Body (%w)", err2))
		}
	}()

	var resBytes []byte
	resBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not ReadAll bytes from Response Body (%w)", err)
	}

	if len(resBytes) <= 1 {
		return nil, nil
	}

	vs := strings.Split(string(resBytes[:len(resBytes)-1]), "\n")
	versions := make([]string, len(vs))
	copy(versions, vs)

	return versions, nil
}

type Info struct {
	//Version of Module
	Version string `json:"version"`
	//Time module was committed
	Time time.Time `json:"time"`
}

var ErrModuleNotFound = errors.New("modproxy: module not found")

type ErrProxyNotOK struct {
	StatusCode int
	Err        string
}

func (e ErrProxyNotOK) Error() string {
	return e.Err
}

// GetLatestVersion of module defined by modulePath
func (p ModProxy) GetLatestVersion(ctx context.Context, modulePath string) (Info, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.Endpoint+"/"+modulePath+"/@latest", nil)
	if err != nil {
		return Info{}, fmt.Errorf("could not create new Request (%w)", err)
	}

	var resp *http.Response
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return Info{}, fmt.Errorf("could not Do HTTP Request (%w)", err)
	}

	defer func() {
		if err2 := resp.Body.Close(); err2 != nil {
			err = multierror.Append(err, fmt.Errorf("could not close Response Body (%w)", err2))
		}
	}()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return Info{}, ErrModuleNotFound
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return Info{}, fmt.Errorf("could not ReadAll from Response Body (%w)", err)
		}

		return Info{}, ErrProxyNotOK{StatusCode: resp.StatusCode, Err: string(b)}
	}

	decoder := json.NewDecoder(resp.Body)

	var info Info
	err = decoder.Decode(&info)
	if err != nil {
		return Info{}, fmt.Errorf("could not Json decode Response from /@latest goproxy endpoint (%w)", err)
	}

	err = resp.Body.Close()
	if err != nil {
		return Info{}, fmt.Errorf("could not close Response Body (%w)", err)
	}

	return info, nil
}

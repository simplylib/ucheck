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
)

type Module struct {
	Path     string
	Versions []string
} // todo: add GetModule() with all information

type ModProxy struct {
	Endpoint string
}

// ListVersions of a modulePath as an unsorted []string, []string is nil when there are no versions
func (p ModProxy) ListVersions(ctx context.Context, modulePath string) (versions []string, err error) {
	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, p.Endpoint+"/"+modulePath+"/@v/list", nil)
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
			err = errors.Join(err, fmt.Errorf("could not close Response Body (%w)", err2))
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
	versions = make([]string, len(vs))
	copy(versions, vs)

	return versions, nil
}

type Info struct {
	// Version of Module
	Version string `json:"version"`
	// Time module was committed
	Time time.Time `json:"time"`
}

var ErrModuleNotFound = errors.New("modproxy: module not found")

type ProxyNotOKError struct {
	StatusCode int
	Err        string
}

func (e ProxyNotOKError) Error() string {
	return e.Err
}

// GetLatestVersion of module defined by modulePath
func (p ModProxy) GetLatestVersion(ctx context.Context, modulePath string) (info Info, err error) {
	var req *http.Request
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, p.Endpoint+"/"+modulePath+"/@latest", nil)
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
			err = errors.Join(err, fmt.Errorf("could not close Response Body (%w)", err2))
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

		return Info{}, ProxyNotOKError{StatusCode: resp.StatusCode, Err: string(b)}
	}

	decoder := json.NewDecoder(resp.Body)

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

const DefaultGoProxy = "https://proxy.golang.org"

func ListVersions(ctx context.Context, modulePath string) ([]string, error) {
	return ModProxy{Endpoint: DefaultGoProxy}.ListVersions(ctx, modulePath)
}

func GetLatestVersion(ctx context.Context, modulePath string) (Info, error) {
	return ModProxy{Endpoint: DefaultGoProxy}.GetLatestVersion(ctx, modulePath)
}

package modproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/mod/semver"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"
)

func getVersionsFromGoBinary(modulePath string) ([]string, error) {
	cmd := exec.Command("go", "list", "-m", "-versions", "-json", modulePath)
	cmd.Stderr = &bytes.Buffer{}
	cmd.Stdout = &bytes.Buffer{}

	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	outputJson := struct {
		Versions []string `json:"versions"`
		Error    *struct {
			Err string `json:"Err"`
		}
	}{}

	err = json.Unmarshal(cmd.Stdout.(*bytes.Buffer).Bytes(), &outputJson)
	if err != nil {
		return nil, err
	}

	if outputJson.Error != nil {
		return nil, fmt.Errorf(outputJson.Error.Err)
	}

	return outputJson.Versions, nil
}

func TestModProxy_ListVersions(t *testing.T) {
	tests := []struct {
		modulePath string
		want       []string
		wantErr    bool
	}{
		{
			modulePath: "golang.org/x/mod",
			want:       []string{"v0.3.0", "v0.5.0", "v0.4.0"},
			wantErr:    false,
		},
		{
			modulePath: "golang.org/x/sys",
			want:       nil,
			wantErr:    false,
		},
	}

	var modProxy ModProxy
	if testing.Short() {
		serv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			modulePath := strings.TrimPrefix(strings.TrimSuffix(req.URL.Path, "/@v/list"), "/")
			for _, tt := range tests {
				if tt.modulePath != modulePath {
					continue
				}
				_, err := io.WriteString(rw, strings.Join(tt.want, "\n")+"\n")
				if err != nil {
					t.Fatal(err)
					return
				}
			}
		}))
		defer serv.Close()
		modProxy.Endpoint = serv.URL
	} else {
		modProxy.Endpoint = "https://proxy.golang.org"
		var err error
		for i, tt := range tests {
			tests[i].want, err = getVersionsFromGoBinary(tt.modulePath)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	for _, tt := range tests {
		t.Run(tt.modulePath, func(t *testing.T) {
			got, err := modProxy.ListVersions(context.Background(), tt.modulePath)
			if tt.wantErr && err == nil {
				t.Fatal("wanted err, got nil")
			}
			if !testing.Short() {
				semver.Sort(got)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("slices not equal, got (%v) want (%v)", got, tt.want)
			}
		})
	}
}

func getLatestVersionFromGoBinary(modulePath string) (Info, error) {
	//#nosec G204
	cmd := exec.Command("go", "list", "-m", "-json", modulePath+"@latest")
	cmd.Stderr = &bytes.Buffer{}
	cmd.Stdout = &bytes.Buffer{}

	err := cmd.Run()
	if err != nil {
		return Info{}, err
	}

	outputJson := struct {
		Version string    `json:"Version"`
		Time    time.Time `json:"Time"`
		Error   *struct {
			Err string `json:"Err"`
		}
	}{}

	err = json.Unmarshal(cmd.Stdout.(*bytes.Buffer).Bytes(), &outputJson)
	if err != nil {
		return Info{}, err
	}

	if outputJson.Error != nil {
		return Info{}, fmt.Errorf(outputJson.Error.Err)
	}

	return Info{Version: outputJson.Version, Time: outputJson.Time}, nil
}

func TestModProxy_GetLatestVersion(t *testing.T) {
	tests := []struct {
		name       string
		modulePath string
		modules    []Module
		want       Info
		wantErr    bool
	}{
		{
			name:       "basic",
			modulePath: "golang.org/x/mod",
			want: Info{
				Version: "v0.5.1",
				Time:    time.Unix(1000, 1000),
			},
			modules: []Module{
				{
					Path:     "golang.org/x/mod",
					Versions: []string{},
				},
			},
			wantErr: false,
		},
		{
			name:       "not exist",
			modulePath: "golang.org/x/no_mod",
			want:       Info{},
			modules:    []Module{},
			wantErr:    true,
		},
	}

	var currentTest int
	var modProxy ModProxy
	if testing.Short() {
		serv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			modulePath := strings.TrimPrefix(strings.TrimSuffix(req.URL.Path, "/@latest"), "/")
			encoder := json.NewEncoder(rw)
			for _, m := range tests[currentTest].modules {
				if m.Path != modulePath {
					continue
				}
				err := encoder.Encode(tests[currentTest].want)
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			rw.WriteHeader(http.StatusNotFound)
		}))
		defer serv.Close()
		modProxy.Endpoint = serv.URL
	} else {
		modProxy.Endpoint = "https://proxy.golang.org"
		var err error
		for i, tt := range tests {
			tests[i].want, err = getLatestVersionFromGoBinary(tt.modulePath)
			if err != nil && !tests[i].wantErr {
				t.Fatal(err)
			}
		}
	}

	tt := struct {
		name       string
		modulePath string
		modules    []Module
		want       Info
		wantErr    bool
	}{}
	for currentTest, tt = range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
			defer cancelFunc()
			got, err := modProxy.GetLatestVersion(ctx, tt.modulePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.want.Time.Equal(got.Time) || tt.want.Version != got.Version {
				t.Errorf("GetLatestVersion() got = %v, want %v", got, tt.want)
			}
		})
	}
}

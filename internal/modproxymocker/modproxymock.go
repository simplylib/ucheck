package modproxymocker

import (
	"context"

	"github.com/simplylib/ucheck/godep/modproxy"
)

type MockModProxy struct {
	Modules []modproxy.Module
}

func (mp *MockModProxy) ListVersions(ctx context.Context, modulePath string) ([]string, error) {
	for _, m := range mp.Modules {
		if m.Path == modulePath {
			return m.Versions, nil
		}
	}
	return nil, nil
}

func (mp *MockModProxy) GetLatestVersion(ctx context.Context, modulePath string) (modproxy.Info, error) {
	for _, m := range mp.Modules {
		if m.Path == modulePath {
			return modproxy.Info{Version: m.Versions[len(m.Versions)-1]}, nil
		}
	}
	return modproxy.Info{}, nil
}

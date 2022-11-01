package godep

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/simplylib/ucheck/godep/modproxy"
	"github.com/simplylib/ucheck/internal/modproxymocker"
)

// todo: add short tests for no server interaction
func TestCheckGoModBytesForUpdates(t *testing.T) {
	tests := []struct {
		name     string
		modBytes []byte
		modules  []modproxy.Module
		want     []Update
		wantErr  bool
	}{
		{
			name: "has update",
			modBytes: []byte(
				`module gitea.ruxion.com/Ruxion/repobot

go 1.18

require golang.org/x/mod v0.5.0`),
			modules: []modproxy.Module{
				{
					Path: "golang.org/x/mod",
					Versions: []string{
						"v0.5.0",
						"v0.5.1",
						"v0.5.2",
					},
				},
			},
			want: []Update{
				{
					module:     "golang.org/x/mod",
					oldVersion: "v0.5.0",
					newVersion: "v0.5.2",
				},
			},
			wantErr: false,
		},
		{
			name: "no update",
			modBytes: []byte(
				`module gitea.ruxion.com/Ruxion/repobot

go 1.18

require golang.org/x/mod v0.5.1`),
			modules: []modproxy.Module{
				{
					Path: "golang.org/x/mod",
					Versions: []string{
						"v0.5.0",
						"v0.5.1",
					},
				},
			},
			want:    nil,
			wantErr: false,
		},
	}

	goModProxy := &modproxymocker.MockModProxy{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goModProxy.Modules = tt.modules
			ctx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
			defer cancelFunc()
			got, err := CheckGoModBytesForUpdates(ctx, goModProxy, tt.modBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckGoModBytesForUpdates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckGoModBytesForUpdates() got = %v, want %v", got, tt.want)
			}
		})
	}
}

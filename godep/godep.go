package godep

import (
	"context"
	"fmt"
	"sync"

	"github.com/simplylib/errgroup"
	"github.com/simplylib/ucheck/modproxy"
	"golang.org/x/mod/modfile"
)

type ModProxy interface {
	GetLatestVersion(ctx context.Context, modulePath string) (modproxy.Info, error)
}

type Update struct {
	Module     string
	OldVersion string
	NewVersion string
}

type GoDep struct {
	// Proxy to use for checking Versions
	Proxy ModProxy

	// MaxRequests is the limit *per function call* of how many http requests to send at once
	MaxRequests int
}

// CheckGoModBytesForUpdates and return slice of Avaliable Updates
func (gp *GoDep) CheckGoModBytesForUpdates(ctx context.Context, b []byte) ([]Update, error) {
	file, err := modfile.Parse("go.mod", b, nil)
	if err != nil {
		return nil, fmt.Errorf("could not parse mod bytes (%w)", err)
	}
	requires := file.Require
	if len(requires) == 0 {
		return nil, nil
	}

	var eg errgroup.Group
	eg.SetLimit(gp.MaxRequests)

	var (
		updates []Update
		mu      sync.Mutex
	)
	for _, require := range requires {
		require := require
		eg.Go(func() error {
			info, err := gp.Proxy.GetLatestVersion(ctx, require.Mod.Path)
			if err != nil {
				return fmt.Errorf("could not get latest version of (%v) from proxy due to error (%w)", require.Mod.Path, err)
			}

			if info.Version == require.Mod.Version {
				return nil
			}

			mu.Lock()
			updates = append(updates, Update{
				Module:     require.Mod.Path,
				OldVersion: require.Mod.Version,
				NewVersion: info.Version,
			})
			mu.Unlock()

			return nil
		})
	}

	err = eg.Wait()
	if err != nil {
		return nil, err
	}

	return updates, nil
}

// CheckGoModBytesForUpdates returns a slice of Update's available in passed modBytes
// Deprecated: this is now replaced with a shim to calling the same function as a method on GoDep
func CheckGoModBytesForUpdates(ctx context.Context, proxy ModProxy, modBytes []byte) ([]Update, error) {
	return (&GoDep{Proxy: proxy, MaxRequests: 1}).CheckGoModBytesForUpdates(ctx, modBytes)
}

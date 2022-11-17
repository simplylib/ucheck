package godep

import (
	"context"
	"fmt"

	"github.com/simplylib/ucheck/modproxy"
	"golang.org/x/mod/modfile"
)

type ModProxy interface {
	GetLatestVersion(ctx context.Context, modulePath string) (modproxy.Info, error)
}

type Update struct {
	module     string
	oldVersion string
	newVersion string
}

// CheckGoModBytesForUpdates returns a slice of Update's available in passed modBytes
func CheckGoModBytesForUpdates(ctx context.Context, proxy ModProxy, modBytes []byte) ([]Update, error) {
	file, err := modfile.Parse("go.mod", modBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("could not parse mod bytes (%w)", err)
	}
	requires := file.Require
	if len(requires) == 0 {
		return nil, nil
	}

	var updates []Update
	var info modproxy.Info
	for _, require := range requires {
		info, err = proxy.GetLatestVersion(ctx, require.Mod.Path)
		if err != nil {
			return nil, fmt.Errorf("could not get latest version of (%v) from proxy due to error (%w)", require.Mod.Path, err)
		}
		if info.Version == require.Mod.Version {
			continue
		}
		updates = append(updates, Update{
			module:     require.Mod.Path,
			oldVersion: require.Mod.Version,
			newVersion: info.Version,
		})
	}

	return updates, nil
}

package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/simplylib/ucheck/godep"
	"github.com/simplylib/ucheck/modproxy"
)

func run() error {
	goProxy := flag.String("goproxy", "https://proxy.golang.org", "base url of go proxy server")

	flag.CommandLine.Usage = func() {
		fmt.Fprintln(
			flag.CommandLine.Output(),
			"Usage: "+os.Args[0]+" <flags> <project dir(s)>\n",
		)
		fmt.Fprintln(
			flag.CommandLine.Output(),
			"Project directory is optional and can be multiple directories seperated by a space, defaults to current directory",
		)
		fmt.Fprintln(
			flag.CommandLine.Output(),
			"\nFlags: ",
		)
		flag.CommandLine.PrintDefaults()
	}

	flag.Parse()

	var paths []string
	if flag.NArg() == 0 {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get the current directory (%w)", err)
		}

		paths = []string{wd}
	} else {
		paths = flag.Args()
	}

	var (
		err     error
		updates []godep.Update
		buf     []byte
	)
	for i := range paths {
		buf, err = os.ReadFile(filepath.Join(paths[i], string(filepath.Separator)+"go.mod"))
		if err != nil {
			return fmt.Errorf("could not read file (%v) error (%w)", paths[i], err)
		}

		updates, err = godep.CheckGoModBytesForUpdates(context.Background(), modproxy.ModProxy{Endpoint: *goProxy}, buf)
		if err != nil {
			return fmt.Errorf("could not check (%v) for updates due to error (%w)", filepath.Join(paths[i], string(filepath.Separator)+"go.mod"), err)
		}

		if len(updates) != 0 {
			log.Printf("path (%v) has updates\n", paths[i])
		}
	}

	return nil
}

func Main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

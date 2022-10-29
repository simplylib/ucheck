package cmd

import "log"

func run() error {
	return nil
}

func Main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

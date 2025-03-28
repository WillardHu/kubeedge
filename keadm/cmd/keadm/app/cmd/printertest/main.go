package main

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/kubeedge/kubeedge/keadm/cmd/keadm/app/cmd/util/printer"
)

func main() {
	example()
}

func example() {
	p := printer.NewColorPrinter("join")
	p.SetDebug(true)

	var first, done printer.StepDone
	input, err := p.Input("Are you sure you want to run join? [y/N]:")
	if err != nil {
		p.Error(err, "Failed to obtain user input")
		os.Exit(1)
	}
	if strings.ToLower(input) != "y" {
		p.Warn("The user aborted the join operation ...")
		return
	}

	first = p.Step("Check KubeEdge edgecore process status")
	// check process ...
	first()

	done = p.Step("Check if the management directory is clean")
	// check process ...
	done()

	done = p.Step("Check if the node name is valid")
	// check process...
	done()

	done = p.Step("Create the necessary directories")
	// create process...
	time.Sleep(500 * time.Millisecond)
	done()

	err = p.StepAndRun(func() error {
		// pull image process ...
		p.Debug("Pulling docker.m.daocloud.io/kubeedge/installation-package:v1.20.0 ...")
		time.Sleep(1 * time.Second)
		p.Debug("Successfully pulled docker.m.daocloud.io/kubeedge/installation-package:v1.20.0")
		return nil
	}, "Pull images")
	if err != nil {
		p.Error(err, "Failed to pull images")
		os.Exit(1)
	}

	done = p.Step("Copy resources from the image to the management directory")
	// copy resources process ...
	time.Sleep(1 * time.Second)
	done()

	done = p.Step("Generate systemd service file")
	done()

	done = p.Step("Generate EdgeCore default configuration")
	done()

	p.Info("The configuration does not exist or the parsing fails, and the default configuration is generated")
	p.Warn("NodeIP is empty , use default ip which can connect to cloud")
	done = p.Step("Start edgecore")
	done()

	p.Info("KubeEdge edgecore is running, For logs visit: journalctl -u edgecore.service -xe")

	p.Info("Test failed message")
	p.Error(errors.New("failed to start edgecore"), "Edgenode join failed.")
	first()
}

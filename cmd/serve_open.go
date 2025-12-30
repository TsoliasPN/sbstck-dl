package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
)

func openBrowser(url string) error {
	cmd, err := browserCommandForOS(runtime.GOOS, url)
	if err != nil {
		return err
	}
	return cmd.Start()
}

func browserCommandForOS(goos string, url string) (*exec.Cmd, error) {
	switch goos {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", url), nil
	case "darwin":
		return exec.Command("open", url), nil
	case "linux":
		return exec.Command("xdg-open", url), nil
	default:
		return nil, fmt.Errorf("unsupported platform %q", goos)
	}
}

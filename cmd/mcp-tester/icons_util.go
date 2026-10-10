package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// iconOut is where icon checks report: stderr with --format json, so that
// stdout stays pure JSON.
func iconOut() io.Writer {
	if format == "json" {
		return os.Stderr
	}
	return os.Stdout
}

func checkAndDownloadIcons(icons []mcp.Icon, downloadDir string) {
	if len(icons) == 0 {
		return
	}
	for _, icon := range icons {
		fmt.Fprintf(iconOut(), "  Checking Icon: %s\n", icon.Source)
		if strings.HasPrefix(icon.Source, "data:") {
			fmt.Fprintln(iconOut(), "    - Info: Data URI (embedded base64)")
			continue
		}

		resp, err := http.Head(icon.Source)
		if err != nil {
			fmt.Fprintf(iconOut(), "    - Error: Failed to reach icon: %v\n", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			fmt.Fprintf(iconOut(), "    - Error: Icon returned HTTP %d\n", resp.StatusCode)
		} else {
			fmt.Fprintf(iconOut(), "    - Success: Reachable (%s)\n", resp.Header.Get("Content-Type"))
		}

		if downloadDir != "" {
			downloadIcon(icon.Source, downloadDir)
		}
	}
}

func downloadIcon(url, dir string) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Fprintf(iconOut(), "    - Error creating dir: %v\n", err)
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(iconOut(), "    - Download failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	filename := filepath.Base(url)
	if !strings.Contains(filename, ".") {
		filename += ".png" // Default extension
	}
	path := filepath.Join(dir, filename)

	out, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(iconOut(), "    - File creation failed: %v\n", err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Fprintf(iconOut(), "    - Save failed: %v\n", err)
		return
	}
	fmt.Fprintf(iconOut(), "    - Saved to: %s\n", path)
}

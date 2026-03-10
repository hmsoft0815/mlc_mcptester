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

func checkAndDownloadIcons(icons []mcp.Icon, downloadDir string) {
	if len(icons) == 0 {
		return
	}
	for _, icon := range icons {
		fmt.Printf("  Checking Icon: %s\n", icon.Source)
		if strings.HasPrefix(icon.Source, "data:") {
			fmt.Println("    - Info: Data URI (embedded base64)")
			continue
		}

		resp, err := http.Head(icon.Source)
		if err != nil {
			fmt.Printf("    - Error: Failed to reach icon: %v\n", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			fmt.Printf("    - Error: Icon returned HTTP %d\n", resp.StatusCode)
		} else {
			fmt.Printf("    - Success: Reachable (%s)\n", resp.Header.Get("Content-Type"))
		}

		if downloadDir != "" {
			downloadIcon(icon.Source, downloadDir)
		}
	}
}

func downloadIcon(url, dir string) {
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Printf("    - Error creating dir: %v\n", err)
		return
	}

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("    - Download failed: %v\n", err)
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
		fmt.Printf("    - File creation failed: %v\n", err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Printf("    - Save failed: %v\n", err)
		return
	}
	fmt.Printf("    - Saved to: %s\n", path)
}

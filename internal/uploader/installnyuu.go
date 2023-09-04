package uploader

import (
	"archive/tar"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/xi2/xz"
)

func DownloadNyuuRelease(version string, path string) error {
	// Determine the current system architecture
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "amd64"
	case "win32":
		arch = "win32"
	case "x64":
		arch = "x64"
	case "aarch64":
		arch = "aarch64"
	case "arm64":
		fmt.Printf("WARNING: arm64 arch detected, falback to aarch64. The upload might won't work.\n")
		arch = "aarch64"
	default:
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	// Construct the URL for the ngPost release based on the current system architecture
	url := fmt.Sprintf("https://github.com/animetosho/Nyuu/releases/download/v%s/nyuu-v%s-linux-%s.tar.xz", version, version, arch)

	log.Default().Printf("Downloading nyuu release %s...", url)

	// Send a GET request to the ngPost release URL
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download nyuu release: %v", err)
	}
	defer resp.Body.Close()

	// Create a new file to write the downloaded release to
	file, err := os.Create(path + ".tar.xz")
	if err != nil {
		return fmt.Errorf("failed to create nyuu file: %v", err)
	}
	defer file.Close()

	// Write the downloaded release to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write nyuu release to file: %v", err)
	}

	// Open the downloaded file for reading
	f, err := os.Open(path + ".tar.xz")
	if err != nil {
		return fmt.Errorf("failed to open nyuu file: %v", err)
	}
	defer f.Close()

	uncompressedStream, err := xz.NewReader(f, 0)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %v", err)
	}
	// Create a new tar reader for the xz reader
	tr := tar.NewReader(uncompressedStream)

	// Iterate over the files in the tar archive
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %v", err)
		}

		// If the current file is the nyuu binary, extract it and copy it to the given path
		if hdr.Name == "nyuu" {
			// Create a new file to write the nyuu binary to
			out, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("failed to create nyuu binary file: %v", err)
			}
			defer out.Close()

			// Copy the nyuu binary to the file
			_, err = io.Copy(out, tr)
			if err != nil {
				return fmt.Errorf("failed to write nyuu binary to file: %v", err)
			}

			// Set the file permissions to 0755
			err = os.Chmod(path, 0755)
			if err != nil {
				return fmt.Errorf("failed to change file permissions: %v", err)
			}

			// Exit the loop since we've found and extracted the nyuu binary
			break
		}
	}

	// Remove the downloaded tar.xz file
	err = os.Remove(path + ".tar.xz")
	if err != nil {
		return fmt.Errorf("failed to remove nyuu tar file: %v", err)
	}

	return nil
}

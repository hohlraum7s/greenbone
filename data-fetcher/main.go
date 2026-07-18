package main

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	authToken       = os.Getenv("FETCH_AUTH_TOKEN")
	port            = os.Getenv("PORT")
	destVtData      = "/dest/vt-data"
	destScapData    = "/dest/scap-data"
	destCertData    = "/dest/cert-data"
	destGvmdData    = "/dest/gvmd-data/data-objects"
	destNotusData   = "/dest/notus-data"
)

func main() {
	if authToken == "" {
		log.Println("WARNING: FETCH_AUTH_TOKEN is not set. API will be unauthenticated!")
	}
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/upload", handleUpload)
	http.HandleFunc("/healthz", handleHealthz)

	log.Printf("Starting GVM data-fetcher API server on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed. Use POST.", http.StatusMethodNotAllowed)
		return
	}

	// Verify authorization token
	if authToken != "" {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized: Missing Bearer token", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token != authToken {
			http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
			return
		}
	}

	log.Println("Upload request authorized. Parsing feed archive...")

	// Limit multipart form size to 10GB for large feeds
	err := r.ParseMultipartForm(10 << 30)
	if err != nil {
		log.Printf("Failed to parse multipart form: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse multipart form: %v", err), http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		log.Printf("Failed to get file from form: %v", err)
		http.Error(w, "Missing 'file' field in multipart form.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Initialize gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		log.Printf("Failed to initialize gzip reader: %v", err)
		http.Error(w, "Invalid gzip format.", http.StatusBadRequest)
		return
	}
	defer gzReader.Close()

	// Initialize tar reader
	tarReader := tar.NewReader(gzReader)
	filesExtracted := 0

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			log.Printf("Error reading tar entry: %v", err)
			http.Error(w, fmt.Sprintf("Error reading archive: %v", err), http.StatusInternalServerError)
			return
		}

		// Clean path to prevent directory traversal
		cleanPath := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanPath, "../") || strings.HasPrefix(cleanPath, "/") {
			log.Printf("Warning: Skipped unsafe path: %s", header.Name)
			continue
		}

		// Determine destination base folder by splitting path components
		parts := strings.Split(cleanPath, string(filepath.Separator))
		if len(parts) < 2 {
			// Skip top-level files or empty entries
			continue
		}

		firstDir := parts[0]
		subPath := filepath.Join(parts[1:]...)

		var targetBase string
		switch firstDir {
		case "plugins":
			targetBase = destVtData
		case "scap-data":
			targetBase = destScapData
		case "cert-data":
			targetBase = destCertData
		case "data-objects":
			targetBase = destGvmdData
		case "notus":
			targetBase = destNotusData
		default:
			// Unrecognized directory, skip it
			continue
		}

		targetPath := filepath.Join(targetBase, subPath)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				log.Printf("Failed to create directory %s: %v", targetPath, err)
				http.Error(w, "Failed to create directory.", http.StatusInternalServerError)
				return
			}

		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				log.Printf("Failed to create parent directory for %s: %v", targetPath, err)
				http.Error(w, "Failed to create directories.", http.StatusInternalServerError)
				return
			}

			// Create and copy file contents
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				log.Printf("Failed to create file %s: %v", targetPath, err)
				http.Error(w, "Failed to write files to persistent volumes.", http.StatusInternalServerError)
				return
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				log.Printf("Failed to copy content to file %s: %v", targetPath, err)
				http.Error(w, "Failed to copy file contents.", http.StatusInternalServerError)
				return
			}
			outFile.Close()
			filesExtracted++
		}
	}

	log.Printf("Import completed successfully! Extracted %d files.", filesExtracted)
	w.WriteHeader(http.StatusOK)
	w.Write(fmt.Appendf(nil, "Import successful. Extracted %d files.", filesExtracted))
}

package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
)

const (
	inputDir     = "./content" // Directory containing markdown files
	outputDir    = "./public"  // Directory to output HTML files
	cssSourceDir = "./css"     // Directory containing the CSS files
	cssFile      = "main.css"  // Primitive CSS file to be copied
	cssDestDir   = "css"       // Destination directory within the output directory
)

func main() {
	// Generate the site
	generateSite()

	// Serve the generated site
	serveSite()
}

func generateSite() {
	// Create the output directory if it doesn't exist
	err := os.MkdirAll(outputDir, os.ModePerm)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Copy the CSS file to the output directory
	err = copyCSSFile()
	if err != nil {
		log.Fatalf("Failed to copy CSS file: %v", err)
	}

	// Generate the site with navigation
	err = filepath.Walk(inputDir, processFile)
	if err != nil {
		log.Fatalf("Error walking the path %q: %v", inputDir, err)
	}

	fmt.Println("Site generated successfully.")
}

func serveSite() {
	// Serve files from the outputDir
	fs := http.FileServer(http.Dir(outputDir))
	http.Handle("/", fs)

	// Start the server on port 8080
	fmt.Println("Serving at http://localhost:8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// processFile is called for each file found by filepath.Walk
func processFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	// Skip directories
	if info.IsDir() {
		return nil
	}

	// Process only markdown files
	if strings.HasSuffix(info.Name(), ".md") {
		err = convertMarkdownToHTML(path)
		if err != nil {
			log.Printf("Failed to convert %s: %v", path, err)
		}
	}

	return nil
}

// convertMarkdownToHTML converts a markdown file to HTML and saves it
func convertMarkdownToHTML(mdPath string) error {
	// Read the markdown file
	mdContent, err := ioutil.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Convert markdown to HTML using goldmark
	var htmlContent strings.Builder
	md := goldmark.New()
	err = md.Convert(mdContent, &htmlContent)
	if err != nil {
		return fmt.Errorf("failed to convert markdown to HTML: %w", err)
	}

	// Generate navigation bar
	navBar := generateNavBar()

	// Wrap content with <div class="medium-container">
	finalHTML := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <link rel="stylesheet" href="/%s/%s">
</head>
<body>
    %s
    <div class="medium-container">
        %s
    </div>
</body>
</html>
`, filepath.Base(mdPath), cssDestDir, cssFile, navBar, htmlContent.String())

	// Determine output path
	relPath, err := filepath.Rel(inputDir, mdPath)
	if err != nil {
		return fmt.Errorf("failed to determine relative path: %w", err)
	}

	htmlPath := filepath.Join(outputDir, strings.Replace(relPath, ".md", ".html", 1))

	// Ensure output directory exists
	err = os.MkdirAll(filepath.Dir(htmlPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Write the final HTML content to the output file
	err = ioutil.WriteFile(htmlPath, []byte(finalHTML), 0644)
	if err != nil {
		return fmt.Errorf("failed to write HTML file: %w", err)
	}

	return nil
}

// copyCSSFile copies the CSS file from the source directory to the output directory
func copyCSSFile() error {
	srcPath := filepath.Join(cssSourceDir, cssFile)
	destPath := filepath.Join(outputDir, cssDestDir, cssFile)

	// Ensure the destination directory exists
	err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create CSS destination directory: %w", err)
	}

	// Copy the file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open CSS source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create CSS destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy CSS file: %w", err)
	}

	return nil
}

// generateNavBar generates a navigation bar based on the markdown files and directories
func generateNavBar() string {
	var navBar strings.Builder

	navBar.WriteString(`<div class="medium-container"><ul style="list-style: none; display: flex; gap: 10px;">`)

	// Walk through the directory and create navigation links
	filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != inputDir {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".md") {
			relPath, err := filepath.Rel(inputDir, path)
			if err != nil {
				return err
			}
			htmlFileName := strings.Replace(relPath, ".md", ".html", 1)
			link := fmt.Sprintf(`<li><a href="/%s">%s</a></li>`, htmlFileName, strings.TrimSuffix(filepath.Base(info.Name()), ".md"))
			navBar.WriteString(link)
		}

		return nil
	})

	navBar.WriteString(`</ul></div>`)
	return navBar.String()
}

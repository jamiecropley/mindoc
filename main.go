package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// Page represents an HTML page
type Page struct {
	Title    string
	Content  string
	Dir      string
	Filename string
}

// Configurations
const (
	inputDir  = "docs"
	outputDir = "site"
	indexFile = "index.html"
	imgDir    = "img" // Folder for images within the docs directory
)

// Template for HTML files
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="preconnect" href="https://rsms.me/">
    <link rel="stylesheet" href="https://rsms.me/inter/inter.css">
    <style>
        :root {
            font-family: Inter, sans-serif;
            font-feature-settings: 'liga' 1, 'calt' 1; /* fix for Chrome */
            color: #404040;
        }
        @supports (font-variation-settings: normal) {
            :root { font-family: InterVariable, sans-serif; }
        }
        body {
            padding: 20px;
        }
        h1, h2, h3, h4, h5, h6 {
            color: #404040;
        }
        a {
            color: #404040;
			text-decoration: none;
        }

		a:hover {
			font-weight: bold;
		}
    </style>
</head>
<body>
    {{.Content}}
</body>
</html>`

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Table of Contents</title>
    <link rel="preconnect" href="https://rsms.me/">
    <link rel="stylesheet" href="https://rsms.me/inter/inter.css">
    <style>
        :root {
            font-family: Inter, sans-serif;
            font-feature-settings: 'liga' 1, 'calt' 1; /* fix for Chrome */
            color: #404040;
        }
        @supports (font-variation-settings: normal) {
            :root { font-family: InterVariable, sans-serif; }
        }
        body {
            padding: 20px;
        }
        h1, h2, h3, h4, h5, h6 {
            color: #404040;
        }
        a {
            color: #404040;
			text-decoration: none;
        }

		a:hover {
			font-weight: bold;
		}
    </style>
</head>
<body>
    <h1>Table of Contents</h1>
    <ul>
    {{range $dir, $pages := .}}
        <li>{{ base $dir }}
        <ul>
        {{range $pages}}
            <li><a href="{{.Dir}}/{{.Filename}}">{{.Title}}</a></li>
        {{end}}
        </ul>
        </li>
    {{end}}
    </ul>
</body>
</html>`

func main() {
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	inputPath := filepath.Join(currentDir, inputDir)

	// Delete the output directory if it exists
	err = os.RemoveAll(outputDir)
	if err != nil {
		fmt.Printf("Error deleting output directory: %v\n", err)
		return
	}

	// Copy img directory to outputDir
	err = copyDir(filepath.Join(inputDir, imgDir), filepath.Join(outputDir, imgDir))
	if err != nil {
		fmt.Printf("Error copying img directory: %v\n", err)
		return
	}

	var pages []Page

	err = filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			relativePath, _ := filepath.Rel(inputPath, path)
			dir := filepath.Dir(relativePath)
			dir = strings.ReplaceAll(dir, " ", "")
			page, err := processFile(path, dir)
			if err != nil {
				return err
			}
			pages = append(pages, page)
		}

		return nil
	})

	if err != nil {
		fmt.Printf("Error processing directory %s: %v\n", inputPath, err)
		os.Exit(1)
	}

	for _, page := range pages {
		err := generateHTML(page)
		if err != nil {
			fmt.Printf("Error generating HTML for page %s: %v\n", page.Title, err)
		}
	}

	err = generateIndex(pages)
	if err != nil {
		fmt.Printf("Error generating index: %v\n", err)
	}
}

func processFile(path string, dir string) (Page, error) {
	input, err := ioutil.ReadFile(path)
	if err != nil {
		return Page{}, err
	}

	title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	title = strings.ReplaceAll(title, " ", "")
	content := convertMarkdownToHTML(input)

	return Page{
		Title:    title,
		Content:  content,
		Dir:      dir,
		Filename: title + ".html",
	}, nil
}

func convertMarkdownToHTML(input []byte) string {
	var buf bytes.Buffer
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(html.WithHardWraps()),
	)
	err := md.Convert(input, &buf)
	if err != nil {
		fmt.Printf("Error converting Markdown to HTML: %v\n", err)
		return ""
	}
	return buf.String()
}

func generateHTML(page Page) error {
	tmpl, err := template.New("page").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(outputDir, page.Dir, page.Filename)
	err = os.MkdirAll(filepath.Join(outputDir, page.Dir), os.ModePerm)
	if err != nil {
		return err
	}

	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	return tmpl.Execute(outputFile, page)
}

func generateIndex(pages []Page) error {
	funcMap := template.FuncMap{
		"base": func(path string) string {
			return filepath.Base(path)
		},
	}

	tmpl, err := template.New("index").Funcs(funcMap).Parse(indexTemplate)
	if err != nil {
		return err
	}

	// Organize pages by directory
	dirMap := make(map[string][]Page)
	for _, page := range pages {
		dirMap[page.Dir] = append(dirMap[page.Dir], page)
	}

	outputFile, err := os.Create(filepath.Join(outputDir, indexFile))
	if err != nil {
		return err
	}
	defer outputFile.Close()

	return tmpl.Execute(outputFile, dirMap)
}

// copyDir copies a whole directory recursively
func copyDir(src string, dst string) error {
	var err error
	var fds []os.FileInfo
	var srcinfo os.FileInfo

	if srcinfo, err = os.Stat(src); err != nil {
		return err
	}

	if err = os.MkdirAll(dst, srcinfo.Mode()); err != nil {
		return err
	}

	if fds, err = ioutil.ReadDir(src); err != nil {
		return err
	}
	for _, fd := range fds {
		srcfp := filepath.Join(src, fd.Name())
		dstfp := filepath.Join(dst, fd.Name())

		if fd.IsDir() {
			if err = copyDir(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		} else {
			if err = copyFile(srcfp, dstfp); err != nil {
				fmt.Println(err)
			}
		}
	}
	return nil
}

// copyFile copies a single file from src to dst
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}

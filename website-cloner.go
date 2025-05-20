package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// Resource types to download
const (
	CSS  = "css"
	JS   = "js"
	IMG  = "img"
	HTML = "html"
)

// Configuration for website cloning
type Config struct {
	URL          string
	OutputDir    string
	MaxDepth     int
	ResourcesDir string
	VisitedURLs  map[string]bool
	mutex        sync.Mutex
	wg           sync.WaitGroup
}

func main() {
	// Parse command line arguments
	urlFlag := flag.String("url", "", "URL of the website to clone")
	outputFlag := flag.String("output", "cloned-site", "Output directory")
	depthFlag := flag.Int("depth", 1, "Maximum depth for crawling links")
	flag.Parse()

	// Check if URL is provided
	if *urlFlag == "" {
		// Check if URL is provided as non-flag argument
		if len(flag.Args()) > 0 {
			*urlFlag = flag.Args()[0]
		} else {
			log.Fatal("Please provide a URL to clone using -url flag or as the first argument")
		}
	}

	// Create configuration
	config := &Config{
		URL:          *urlFlag,
		OutputDir:    *outputFlag,
		MaxDepth:     *depthFlag,
		ResourcesDir: "resources",
		VisitedURLs:  make(map[string]bool),
	}

	// Parse the base URL
	baseURL, err := url.Parse(config.URL)
	if err != nil {
		log.Fatalf("Invalid URL: %v", err)
	}

	// Create output directories
	err = os.MkdirAll(config.OutputDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	resourcesPath := filepath.Join(config.OutputDir, config.ResourcesDir)
	err = os.MkdirAll(filepath.Join(resourcesPath, CSS), 0755)
	if err != nil {
		log.Fatalf("Failed to create CSS directory: %v", err)
	}

	err = os.MkdirAll(filepath.Join(resourcesPath, JS), 0755)
	if err != nil {
		log.Fatalf("Failed to create JS directory: %v", err)
	}

	err = os.MkdirAll(filepath.Join(resourcesPath, IMG), 0755)
	if err != nil {
		log.Fatalf("Failed to create IMG directory: %v", err)
	}

	fmt.Printf("Starting to clone %s into %s\n", config.URL, config.OutputDir)

	// Start cloning process
	config.cloneURL(baseURL, 0)

	// Wait for all goroutines to finish
	config.wg.Wait()

	fmt.Println("Website cloning completed successfully!")
}

// cloneURL downloads a URL and processes its content
func (c *Config) cloneURL(pageURL *url.URL, depth int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Recovered from panic while processing %s: %v\n", pageURL.String(), r)
		}
	}()

	urlStr := pageURL.String()

	// Check if already visited
	c.mutex.Lock()
	if c.VisitedURLs[urlStr] {
		c.mutex.Unlock()
		return
	}
	c.VisitedURLs[urlStr] = true
	c.mutex.Unlock()

	// Only process URLs from the same host
	if pageURL.Host != "" && !strings.Contains(urlStr, strings.TrimPrefix(strings.TrimPrefix(c.URL, "http://"), "https://")) {
		return
	}

	// Get the webpage content
	resp, err := http.Get(urlStr)
	if err != nil {
		fmt.Printf("Failed to fetch %s: %v\n", urlStr, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Got non-200 status code for %s: %d\n", urlStr, resp.StatusCode)
		return
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response body for %s: %v\n", urlStr, err)
		return
	}

	// Determine the output filename
	filename := "index.html"
	if pageURL.Path != "" && pageURL.Path != "/" {
		path := strings.TrimPrefix(pageURL.Path, "/")
		if strings.HasSuffix(path, "/") {
			path = path + "index.html"
		} else if !strings.Contains(filepath.Base(path), ".") {
			path = path + "/index.html"
		}
		filename = path
	}

	outputPath := filepath.Join(c.OutputDir, filename)

	// Create necessary directories
	err = os.MkdirAll(filepath.Dir(outputPath), 0755)
	if err != nil {
		fmt.Printf("Failed to create directory for %s: %v\n", outputPath, err)
		return
	}

	// Parse HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		fmt.Printf("Failed to parse HTML document for %s: %v\n", urlStr, err)
		return
	}

	// Process CSS links
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			c.wg.Add(1)
			go func(href string) {
				defer c.wg.Done()
				c.downloadResource(pageURL, href, CSS)
			}(href)

			// Update href attribute to point to local resource
			localPath := filepath.Join(c.ResourcesDir, CSS, filepath.Base(href))
			s.SetAttr("href", strings.ReplaceAll(localPath, "\\", "/"))
		}
	})

	// Process JavaScript files
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			c.wg.Add(1)
			go func(src string) {
				defer c.wg.Done()
				c.downloadResource(pageURL, src, JS)
			}(src)

			// Update src attribute to point to local resource
			localPath := filepath.Join(c.ResourcesDir, JS, filepath.Base(src))
			s.SetAttr("src", strings.ReplaceAll(localPath, "\\", "/"))
		}
	})

	// Process images
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			c.wg.Add(1)
			go func(src string) {
				defer c.wg.Done()
				c.downloadResource(pageURL, src, IMG)
			}(src)

			// Update src attribute to point to local resource
			localPath := filepath.Join(c.ResourcesDir, IMG, filepath.Base(src))
			s.SetAttr("src", strings.ReplaceAll(localPath, "\\", "/"))
		}
	})

	// Save the modified HTML document
	modifiedHTML, err := doc.Html()
	if err != nil {
		fmt.Printf("Failed to generate HTML for %s: %v\n", urlStr, err)
		return
	}

	err = os.WriteFile(outputPath, []byte(modifiedHTML), 0644)
	if err != nil {
		fmt.Printf("Failed to write HTML file for %s: %v\n", urlStr, err)
		return
	}

	fmt.Printf("Downloaded: %s -> %s\n", urlStr, outputPath)

	// Process links if depth allows
	if depth < c.MaxDepth {
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			if href, exists := s.Attr("href"); exists {
				// Skip external links, anchors, or non-HTTP protocols
				if strings.HasPrefix(href, "#") || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") {
					return
				}

				linkURL, err := url.Parse(href)
				if err != nil {
					return
				}

				// Handle relative URLs
				resolvedURL := pageURL.ResolveReference(linkURL)

				// Only follow links to the same host
				if resolvedURL.Host == pageURL.Host {
					c.wg.Add(1)
					go func(resolvedURL *url.URL, depth int) {
						defer c.wg.Done()
						c.cloneURL(resolvedURL, depth+1)
					}(resolvedURL, depth)
				}
			}
		})
	}
}

// downloadResource downloads a resource file (CSS, JS, IMG) and saves it locally
func (c *Config) downloadResource(baseURL *url.URL, resourceURL string, resourceType string) {
	// Resolve the resource URL
	resolvedURL, err := url.Parse(resourceURL)
	if err != nil {
		fmt.Printf("Failed to parse resource URL %s: %v\n", resourceURL, err)
		return
	}

	// Handle relative URLs
	absoluteURL := baseURL.ResolveReference(resolvedURL)

	// Skip data: URLs
	if strings.HasPrefix(absoluteURL.String(), "data:") {
		return
	}

	// Download the resource
	resp, err := http.Get(absoluteURL.String())
	if err != nil {
		fmt.Printf("Failed to fetch resource %s: %v\n", absoluteURL.String(), err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Got non-200 status code for resource %s: %d\n", absoluteURL.String(), resp.StatusCode)
		return
	}

	// Read the resource data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read resource data for %s: %v\n", absoluteURL.String(), err)
		return
	}

	// Determine the output filename
	filename := filepath.Base(absoluteURL.Path)
	if filename == "" || filename == "." {
		filename = fmt.Sprintf("resource_%d", len(c.VisitedURLs))
	}

	// Ensure the filename is valid
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "&", "_")
	filename = strings.ReplaceAll(filename, "=", "_")

	outputPath := filepath.Join(c.OutputDir, c.ResourcesDir, resourceType, filename)

	// Write the resource file
	err = os.WriteFile(outputPath, data, 0644)
	if err != nil {
		fmt.Printf("Failed to write resource file %s: %v\n", outputPath, err)
		return
	}

	fmt.Printf("Downloaded resource: %s -> %s\n", absoluteURL.String(), outputPath)
}

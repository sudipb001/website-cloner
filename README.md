# website-cloner

## Website Cloner in Go

Here's Go program that clones websites by downloading the HTML content and associated resources (CSS, JavaScript, and images). The program takes a URL as input and creates a local copy of the website.

### Features

- Downloads HTML pages and modifies links to point to local resources
- Downloads and saves CSS, JavaScript, and image files
- Supports crawling links up to a specified depth
- Concurrent downloads using goroutines for better performance
- Handles relative and absolute URLs

### External Dependencies

This program requires one external library:

- **github.com/PuerkitoBio/goquery**: A jQuery-like library for parsing HTML in Go

### Installation Instructions

1. Make sure you have Go installed on your system (version 1.16 or later recommended)

2. Create a new directory for your project and save the code as `main.go`

3. Install the required dependency:

   ```bash
   go mod init website-cloner
   go get github.com/PuerkitoBio/goquery
   ```

4. Build the program:
   For Mac/Linux:

   ```bash
   go build -o website-cloner
   ```

   For Windows:

   ```bash
   go build -o website-cloner.exe
   ```

### Usage

Basic usage:

```bash
./website-cloner https://example.com
```

With optional flags:

```bash
./website-cloner -url https://example.com -output ./cloned-site -depth 2
```

Parameters:

- `-url`: URL of the website to clone (can also be provided as the first argument)
- `-output`: Output directory (default: "cloned-site")
- `-depth`: Maximum depth for crawling links (default: 1)

### How It Works

1. The program parses the provided URL and creates the necessary output directories
2. It downloads the main HTML page and parses it using goquery
3. It identifies all CSS, JavaScript, and image resources and downloads them concurrently
4. It modifies the HTML to point to the local copies of these resources
5. If depth > 0, it follows links to other pages on the same domain and repeats the process

### Limitations

- Only downloads resources from the same domain
- May not handle all types of dynamic content or JavaScript-generated resources
- Some websites may block automated downloads or have restrictions in their robots.txt

// Command checkdocs validates the checked-in API contract and local documentation links.
package main

import (
	"fmt"
	"github.com/goccy/go-yaml"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	raw, err := os.ReadFile("../docs/openapi.yaml")
	if err != nil {
		return err
	}
	var doc map[string]any
	if err = yaml.Unmarshal(raw, &doc); err != nil {
		return err
	}
	refs := 0
	var walk func(any) error
	walk = func(value any) error {
		switch v := value.(type) {
		case map[string]any:
			for key, item := range v {
				if key == "$ref" {
					ref, ok := item.(string)
					if !ok || !strings.HasPrefix(ref, "#/") {
						return fmt.Errorf("unsupported API reference")
					}
					var node any = doc
					for _, part := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
						m, ok := node.(map[string]any)
						if !ok {
							return fmt.Errorf("invalid reference %s", ref)
						}
						part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
						node = m[part]
						if node == nil {
							return fmt.Errorf("missing reference %s", ref)
						}
					}
					refs++
				}
				if err := walk(item); err != nil {
					return err
				}
			}
		case []any:
			for _, item := range v {
				if err := walk(item); err != nil {
					return err
				}
			}
		}
		return nil
	}
	if err = walk(doc); err != nil {
		return err
	}
	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		return fmt.Errorf("API paths missing")
	}
	ids := map[string]bool{}
	operations := 0
	for _, value := range paths {
		path, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid API path")
		}
		for method, value := range path {
			switch method {
			case "get", "post", "patch", "put", "delete":
				operation, ok := value.(map[string]any)
				if !ok {
					return fmt.Errorf("invalid API operation")
				}
				id, ok := operation["operationId"].(string)
				if !ok || id == "" || ids[id] {
					return fmt.Errorf("missing or duplicate operationId")
				}
				ids[id] = true
				operations++
			}
		}
	}
	files, err := filepath.Glob("../docs/*.md")
	if err != nil {
		return err
	}
	files = append(files, "../README.md")
	links := regexp.MustCompile(`\]\(([^)]+)\)`)
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		for _, match := range links.FindAllSubmatch(content, -1) {
			target := string(match[1])
			if strings.HasPrefix(target, "#") || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			target = strings.SplitN(target, "#", 2)[0]
			target, err = url.PathUnescape(target)
			if err != nil {
				return err
			}
			if _, err = os.Stat(filepath.Join(filepath.Dir(file), target)); err != nil {
				return fmt.Errorf("broken local link in %s: %s", file, target)
			}
		}
	}
	fmt.Printf("PASS: %d API operations, %d references, %d documentation files\n", operations, refs, len(files))
	return nil
}

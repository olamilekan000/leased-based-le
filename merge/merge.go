package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	prehelmOutput, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read kustomize build output:", err)
		os.Exit(1)
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting current working directory:", err)
		os.Exit(1)
	}

	filePath := filepath.Join(dir, "charts-all.yaml")

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to open charts-all.yaml:", err)
		os.Exit(1)
	}
	defer file.Close()

	allYamlconfs, err := io.ReadAll(file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read charts-all.yaml:", err)
		os.Exit(1)
	}

	sp, err := splitYAML(prehelmOutput)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to process kustomize output:", err)
		os.Exit(1)
	}

	inputf, err := splitYAML(allYamlconfs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to process all2.yaml:", err)
		os.Exit(1)
	}

	merged := mergeYAMLs(inputf, sp)

	finalOutput := []byte(dumpYAML(merged))

	if _, err := os.Stdout.Write(finalOutput); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to write merged YAML to stdout:", err)
		os.Exit(1)
	}
}

func splitYAML(input []byte) (map[string]string, error) {
	parts := bytes.Split(input, []byte("---"))
	kindToContent := make(map[string]string)

	kindPattern := regexp.MustCompile(`(?m)^kind:\s*(\S+)`)
	namePattern := regexp.MustCompile(`(?m)^\s*name:\s*([^\n]+)`)
	namespacePattern := regexp.MustCompile(`(?m)^\s*namespace:\s*([^\n]+)`)

	for _, part := range parts {
		part = bytes.TrimSpace(part)
		if len(part) == 0 {
			continue
		}

		kindMatch := kindPattern.FindSubmatch(part)
		if kindMatch == nil {
			return nil, fmt.Errorf("failed to extract 'kind' field from:\n%s", string(part))
		}
		kind := string(kindMatch[1])

		nameMatch := namePattern.FindSubmatch(part)
		name := ""
		if nameMatch != nil {
			name = string(bytes.TrimSpace(nameMatch[1]))
		}

		namespaceMatch := namespacePattern.FindSubmatch(part)
		namespace := ""
		if namespaceMatch != nil {
			namespace = string(bytes.TrimSpace(namespaceMatch[1]))
		}

		key := kind
		if name != "" {
			key += "-" + name
		}
		if namespace != "" {
			key += "-" + namespace
		}

		kindToContent[key] = string(part)
	}

	return kindToContent, nil
}

func mergeYAMLs(baseYAML, overrideYAML map[string]string) map[string]string {
	merged := make(map[string]string)

	for kind, content := range baseYAML {
		merged[kind] = content
	}

	for kind, content := range overrideYAML {
		merged[kind] = content
	}

	return merged
}

func dumpYAML(yamlMap map[string]string) string {
	var buffer strings.Builder

	for _, content := range yamlMap {
		buffer.WriteString(content)
		buffer.WriteString("\n---\n")
	}

	return strings.TrimSuffix(buffer.String(), "\n---\n")
}

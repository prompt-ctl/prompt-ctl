package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oleg-koval/promptctl/config"
)

type TemplateMeta struct {
	Current  string   `json:"current"`
	Versions []string `json:"versions"`
}

func runVersion() error {

	if len(os.Args) < 3 {
		return fmt.Errorf("usage: promptctl version <template> [--bump]")
	}

	templateName := os.Args[2]

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	templatePath := filepath.Join(cfg.LocalTemplateDir, templateName+".yaml")
	versionDir := filepath.Join(cfg.LocalTemplateDir, templateName)
	metaPath := filepath.Join(versionDir, "meta.json")

	// Check if already versioned
	if _, err := os.Stat(metaPath); err == nil {
		fmt.Println("Template already versioned.")
		return nil
	}

	// Check flat template exists
	if _, err := os.Stat(templatePath); err != nil {
		return fmt.Errorf("template not found: %s", templateName)
	}

	// Create version folder
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return err
	}

	// Move original to v1.yaml
	v1Path := filepath.Join(versionDir, "v1.yaml")
	if err := os.Rename(templatePath, v1Path); err != nil {
		return err
	}

	meta := TemplateMeta{
		Current:  "v1",
		Versions: []string{"v1"},
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		return err
	}

	fmt.Println("Versioning initialized:")
	fmt.Println("Created v1.yaml and meta.json")

	return nil
}

package cmd

import (
	"fmt"
	"os"

	"github.com/qbits/devdrop/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "devdrop",
	Short:   "Personal development environment CLI",
	Version: version.GetVersion(),
	Long: `DevDrop is a CLI tool that allows developers to create, customize,
and share personal development environments using Docker containers.

Think "dotfiles for entire environments" - portable, version-controlled,
and instantly available anywhere Docker runs.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here
}
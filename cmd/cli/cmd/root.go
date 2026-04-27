package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "file-vault",
	Short: "File Vault CLI — manage your files from the terminal",
	Long:  `A secure file storage CLI that lets you upload, download, and manage your files without leaving the terminal.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

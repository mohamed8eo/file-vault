package cmd

import (
	"github.com/mohamed8eo/file-vault/internal/client"
	"github.com/spf13/cobra"
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "File management commands",
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List all your files",
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.ListFiles()
	},
}

var uploadCmd = &cobra.Command{
	Use:     "upload <path>",
	Short:   "Upload a file",
	Aliases: []string{"up"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.UploadFile(args[0])
	},
}

var getCmd = &cobra.Command{
	Use:     "get <id>",
	Short:   "Get presigned download URL for a file",
	Aliases: []string{"dl"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.GetFile(args[0])
	},
}

var deleteCmd = &cobra.Command{
	Use:     "delete <id>",
	Short:   "Delete a file",
	Aliases: []string{"del", "rm"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.DeleteFile(args[0])
	},
}

func init() {
	filesCmd.AddCommand(listCmd)
	filesCmd.AddCommand(uploadCmd)
	filesCmd.AddCommand(getCmd)
	filesCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(filesCmd)
}

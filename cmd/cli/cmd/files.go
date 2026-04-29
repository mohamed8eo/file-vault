package cmd

import (
	"github.com/mohamed8eo/file-vault/internal/client"
	"github.com/spf13/cobra"
)

var (
	limit   int
	page    int
	offset  int
	sort    string
	fileType string
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
		return client.ListFiles(limit, page, offset, sort, fileType)
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

var searchCmd = &cobra.Command{
	Use:     "search <query>",
	Short:   "Search files by name",
	Aliases: []string{"find"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.SearchFiles(args[0], limit)
	},
}

var downloadCmd = &cobra.Command{
	Use:     "download <id> [output]",
	Short:   "Download file to local machine",
	Aliases: []string{"dl"},
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputPath := ""
		if len(args) > 1 {
			outputPath = args[1]
		}
		return client.DownloadFile(args[0], outputPath)
	},
}

func init() {
	listCmd.Flags().IntVarP(&limit, "limit", "l", 20, "Number of files to list")
	listCmd.Flags().IntVarP(&page, "page", "p", 1, "Page number (1-based)")
	listCmd.Flags().IntVarP(&offset, "offset", "o", 0, "Offset (alternative to page)")
	listCmd.Flags().StringVarP(&sort, "sort", "s", "date", "Sort by: date, name, size")
	listCmd.Flags().StringVarP(&fileType, "type", "t", "", "Filter by type: image, video, document")

	searchCmd.Flags().IntVarP(&limit, "limit", "l", 20, "Max results")

	filesCmd.AddCommand(listCmd)
	filesCmd.AddCommand(uploadCmd)
	filesCmd.AddCommand(getCmd)
	filesCmd.AddCommand(deleteCmd)
	filesCmd.AddCommand(searchCmd)
	filesCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(filesCmd)
}
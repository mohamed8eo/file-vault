package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilesCommandStructure(t *testing.T) {
	assert.Equal(t, "files", filesCmd.Use)
	assert.Equal(t, "File management commands", filesCmd.Short)
}

func TestListCommandStructure(t *testing.T) {
	assert.Equal(t, "list", listCmd.Use)
	assert.Equal(t, "List all your files", listCmd.Short)
	assert.Contains(t, listCmd.Aliases, "ls")
	assert.NotNil(t, listCmd.RunE)
}

func TestUploadCommandStructure(t *testing.T) {
	assert.Equal(t, "upload <path>", uploadCmd.Use)
	assert.Equal(t, "Upload a file", uploadCmd.Short)
	assert.Contains(t, uploadCmd.Aliases, "up")
	assert.NotNil(t, uploadCmd.RunE)
}

func TestGetCommandStructure(t *testing.T) {
	assert.Equal(t, "get <id>", getCmd.Use)
	assert.Equal(t, "Get presigned download URL for a file", getCmd.Short)
	assert.Contains(t, getCmd.Aliases, "dl")
	assert.NotNil(t, getCmd.RunE)
}

func TestDeleteCommandStructure(t *testing.T) {
	assert.Equal(t, "delete <id>", deleteCmd.Use)
	assert.Equal(t, "Delete a file", deleteCmd.Short)
	assert.Contains(t, deleteCmd.Aliases, "del")
	assert.Contains(t, deleteCmd.Aliases, "rm")
	assert.NotNil(t, deleteCmd.RunE)
}

func TestDeleteManyCommandStructure(t *testing.T) {
	assert.Equal(t, "delete-many <id1> <id2> ...", deleteManyCmd.Use)
	assert.Equal(t, "Delete multiple files at once", deleteManyCmd.Short)
	assert.Contains(t, deleteManyCmd.Aliases, "rmall")
	assert.NotNil(t, deleteManyCmd.RunE)
}

func TestStatsCommandStructure(t *testing.T) {
	assert.Equal(t, "stats", statsCmd.Use)
	assert.Equal(t, "Show storage statistics", statsCmd.Short)
	assert.NotNil(t, statsCmd.RunE)
}

func TestSearchCommandStructure(t *testing.T) {
	assert.Equal(t, "search <query>", searchCmd.Use)
	assert.Equal(t, "Search files by name", searchCmd.Short)
	assert.Contains(t, searchCmd.Aliases, "find")
	assert.NotNil(t, searchCmd.RunE)
}

func TestDownloadCommandStructure(t *testing.T) {
	assert.Equal(t, "download <id> [output]", downloadCmd.Use)
	assert.Equal(t, "Download file to local machine", downloadCmd.Short)
	assert.Contains(t, downloadCmd.Aliases, "dl")
	assert.NotNil(t, downloadCmd.RunE)
}
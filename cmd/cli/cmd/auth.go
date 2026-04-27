/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"log"

	"github.com/mohamed8eo/file-vault/internal/client"
	"github.com/spf13/cobra"
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to file vault",
	Aliases: []string{"in"},
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		if provider == "google" || provider == "github" {
			return client.OAuthLogin(provider)
		}
		email, err := cmd.Flags().GetString("email")
		if err != nil {
			log.Fatalf("error: %s\n", err.Error())
		}
		password, err := cmd.Flags().GetString("password")
		if err != nil {
			log.Fatalf("error: %s\n", err.Error())
		}

		return client.Login(email, password)
	},
}

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new account",
	Aliases: []string{"reg"},
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := cmd.Flags().GetString("name")
		if err != nil {
			log.Fatalf("error: %s\n", err.Error())
		}

		email, err := cmd.Flags().GetString("email")
		if err != nil {
			log.Fatalf("error: %s\n", err.Error())
		}

		password, err := cmd.Flags().GetString("password")
		if err != nil {
			log.Fatalf("error: %s\n", err.Error())
		}
		return client.Register(name, email, password)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check login status",
	Aliases: []string{"st"},
	Run: func(cmd *cobra.Command, args []string) {
		client.Status()
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "log out from app",
	Aliases: []string{"out"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.Logout()
	},
}

func init() {
	loginCmd.Flags().StringP("email", "e", "", "your email")
	loginCmd.Flags().StringP("password", "p", "", "your password")
	loginCmd.Flags().StringP("provider", "P", "", "oauth provider (google, github)")

	registerCmd.Flags().StringP("name", "n", "", "your name")
	registerCmd.Flags().StringP("email", "e", "", "your email")
	registerCmd.Flags().StringP("password", "p", "", "your password")
	registerCmd.MarkFlagRequired("name")
	registerCmd.MarkFlagRequired("email")
	registerCmd.MarkFlagRequired("password")

	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(registerCmd)
	authCmd.AddCommand(statusCmd)
	authCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(authCmd)

	fmt.Println()
}

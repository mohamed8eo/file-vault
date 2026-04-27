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
	RunE: func(cmd *cobra.Command, args []string) error {
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
	Run: func(cmd *cobra.Command, args []string) {
		client.Status()
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "log out from app",
	RunE: func(cmd *cobra.Command, args []string) error {
		return client.Logout()
	},
}

func init() {
	loginCmd.Flags().String("email", "", "your email")
	loginCmd.Flags().String("password", "", "your password")
	loginCmd.MarkFlagRequired("email")
	loginCmd.MarkFlagRequired("password")

	registerCmd.Flags().String("name", "", "your name")
	registerCmd.Flags().String("email", "", "your email")
	registerCmd.Flags().String("password", "", "your password")
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

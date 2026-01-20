package main

import (
	"fmt"
	"log"

	"github.com/r4ulcl/nTask/manager"
	"github.com/r4ulcl/nTask/worker"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// @title nTask API
// @version v0.1
// @description nTask API documentation
// @contact.name r4ulcl
// @contact.url https://r4ulcl.com
// @contact.email me@r4ulcl.com
// @license.name  GPL-3.0
// @license.url https://github.com/r4ulcl/nTask/blob/main/LICENSE
// @basePath /
// @schemes https http
// @BasePath /
// @Security ApiKeyAuth
// @securityDefinitions.apikey ApiKeyAuth
// @SecurityScheme ApiKeyAuth
// @in header
// @name Authorization
// @description ApiKeyAuth to login

// Arguments  Config holds configuration parameters.
type Arguments struct {
	ConfigFile      string
	ConfigSSHFile   string
	ConfigCloudFile string
	Swagger         bool
	Verbose         bool
	Debug           bool
	VerifyAltName   bool
}

func main() {
	var arguments Arguments
	version := "v0.2"
	var rootCmd = &cobra.Command{
		Use:     "nTask",
		Short:   "Your program description",
		Version: version, // Set the version here
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return validateGlobalFlags(cmd.Flags(), &arguments)
		},
	}

	// Add global flags to the root command
	rootCmd.PersistentFlags().BoolP("swagger", "s", false, "Start the swagger endpoint (/swagger)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Set verbose mode")
	rootCmd.Flags().BoolP("version", "V", false, "Version for nTask")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Set debug mode")
	rootCmd.PersistentFlags().BoolP("verifyAltName", "a", false, "Set verifyAltName to true")

	// Add manager subcommand
	var managerCmd = &cobra.Command{
		Use:   "manager",
		Short: "Run the manager module",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("Manager:")
			managerStart(&arguments)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateSubcommandFlags(cmd.Flags(), &arguments)
		},
	}

	// Add flags specific to the manager subcommand
	managerCmd.Flags().StringVarP(&arguments.ConfigFile,
		"configFile", "c", "", "Path to the config file (default: manager.conf)")
	managerCmd.Flags().StringVarP(&arguments.ConfigSSHFile,
		"configSSHFile", "f", "", "Path to the config SSH file (default: empty)")
	managerCmd.Flags().StringVarP(&arguments.ConfigCloudFile,
		"configCloudFile", "C", "", "Path to the config Cloud file (default: empty)")

	// Add worker subcommand
	var workerCmd = &cobra.Command{
		Use:   "worker",
		Short: "Run the worker module",
		Run: func(cmd *cobra.Command, args []string) {
			log.Println("Worker:")
			workerStart(&arguments)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return validateSubcommandFlags(cmd.Flags(), &arguments)
		},
	}

	// Add flags specific to the worker subcommand
	workerCmd.Flags().StringVarP(&arguments.ConfigFile,
		"configFile", "c", "", "Path to the config file (default: worker.conf)")

	// Add subcommands to the root command
	rootCmd.AddCommand(managerCmd, workerCmd)

	// Execute the commands
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
	}
}

func managerStart(arguments *Arguments) {
	// Use config parameters to start the manager
	if arguments.ConfigFile == "" {
		arguments.ConfigFile = "manager.conf"
	}
	manager.StartManager(arguments.Swagger, arguments.ConfigFile,
		arguments.ConfigSSHFile, arguments.ConfigCloudFile, arguments.VerifyAltName, arguments.Verbose, arguments.Debug)
}

func workerStart(arguments *Arguments) {
	// Use config parameters to start the worker
	if arguments.ConfigFile == "" {
		arguments.ConfigFile = "worker.conf"
	}
	worker.StartWorker(arguments.Swagger, arguments.ConfigFile,
		arguments.VerifyAltName, arguments.Verbose, arguments.Debug)
}

func validateGlobalFlags(flags *pflag.FlagSet, arguments *Arguments) error {
	var err error
	arguments.Swagger, err = flags.GetBool("swagger")
	if err != nil {
		return fmt.Errorf("error getting 'swagger' flag: %w", err)
	}

	arguments.Verbose, err = flags.GetBool("verbose")
	if err != nil {
		return fmt.Errorf("error getting 'verbose' flag: %w", err)
	}

	arguments.Debug, err = flags.GetBool("debug")
	if err != nil {
		return fmt.Errorf("error getting 'debug' flag: %w", err)
	}

	// If its debug mode its also verbose mode
	if arguments.Debug {
		arguments.Verbose = true
	}

	arguments.VerifyAltName, err = flags.GetBool("verifyAltName")
	return err
}

func validateSubcommandFlags(flags *pflag.FlagSet, arguments *Arguments) error {
	return validateGlobalFlags(flags, arguments)
}

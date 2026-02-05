package main

import (
	"fmt"
	"os"

	"github.com/vladpi/keenetic-routes/app"
	"github.com/vladpi/keenetic-routes/config"

	"github.com/spf13/cobra"
)

func main() {
	var hostFlag, userFlag, passwordFlag string
	service := app.NewService()

	var rootCmd = &cobra.Command{
		Use:     "keenetic-routes",
		Short:   "Manage Keenetic static routes via RCI API",
		Long:    "Upload, backup, and clear static routes on Keenetic routers using the NDMS RCI interface.",
		Version: "1.0.0",
	}

	rootCmd.PersistentFlags().StringVar(&hostFlag, "host", "", "Keenetic router host (e.g., 192.168.100.1:280)")
	rootCmd.PersistentFlags().StringVar(&userFlag, "user", "", "Keenetic router username")
	rootCmd.PersistentFlags().StringVar(&passwordFlag, "password", "", "Keenetic router password")

	loadValidatedConfig := func() (*config.Config, error) {
		cfg, err := config.LoadConfig(hostFlag, userFlag, passwordFlag)
		if err != nil {
			return nil, err
		}
		if err := cfg.Validate(); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	var uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload static routes from a file",
		Long:  "Parse IP/CIDR entries from a file and upload them as static routes to the router.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadValidatedConfig()
			if err != nil {
				return err
			}
			file, _ := cmd.Flags().GetString("file")
			return service.Upload(file, cfg)
		},
	}

	var backupCmd = &cobra.Command{
		Use:   "backup",
		Short: "Backup current static routes to a file",
		Long:  "Download all current static routes from the router and save them to a file in the same format as input files.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadValidatedConfig()
			if err != nil {
				return err
			}
			output, _ := cmd.Flags().GetString("output")
			return service.Backup(output, cfg)
		},
	}

	var clearCmd = &cobra.Command{
		Use:   "clear",
		Short: "Clear all static routes",
		Long:  "Remove all static routes from the router and save configuration.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadValidatedConfig()
			if err != nil {
				return err
			}
			return service.Clear(cfg)
		},
	}

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  "Manage configuration file for Keenetic router connection.",
	}

	var configInitCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize configuration file",
		Long:  "Create a new configuration file interactively.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return service.InitConfig()
		},
	}

	configCmd.AddCommand(configInitCmd)

	uploadCmd.Flags().StringP("file", "f", "", "path to YAML routes file (required)")
	if err := markRequired(uploadCmd, "file"); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	backupCmd.Flags().StringP("output", "o", "", "output YAML file path (required)")
	if err := markRequired(backupCmd, "output"); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	rootCmd.AddCommand(uploadCmd, backupCmd, clearCmd, configCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func markRequired(cmd *cobra.Command, names ...string) error {
	for _, name := range names {
		if err := cmd.MarkFlagRequired(name); err != nil {
			return fmt.Errorf("mark %s required: %w", name, err)
		}
	}
	return nil
}

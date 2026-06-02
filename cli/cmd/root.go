package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/subosito/gotenv"
)

var (
	cfgFile string
	rootCmd = newRootCmd()
)

func Execute() error {
	return rootCmd.Execute()
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "spendsense",
		Short:         "SpendSense CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return err
			}
			return applyFlagOverrides(cmd)
		},
	}

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.expenserc)")
	cmd.PersistentFlags().String("api-url", "", "API server URL (overrides config and env)")

	viper.SetDefault("api_url", "http://localhost:8080")
	viper.SetDefault("base_currency", "USD")
	viper.SetDefault("timezone", "UTC")
	viper.SetDefault("locale", "en-US")

	cmd.AddCommand(newAuthCmd())
	cmd.AddCommand(newExpenseCmd())
	cmd.AddCommand(newCategoryCmd())
	cmd.AddCommand(newWalletCmd())
	cmd.AddCommand(newIncomeCmd())
	cmd.AddCommand(newConfigCmd())

	return cmd
}

func initConfig() error {
	if err := loadEnvFiles(); err != nil {
		return err
	}

	viper.SetEnvPrefix("SPENDSENSE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".expenserc")
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return err
		}
	}

	return nil
}

func loadEnvFiles() error {
	paths := []string{".env"}
	if exePath, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exePath), ".env"))
	}

	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		absolutePath, err := filepath.Abs(path)
		if err == nil {
			path = absolutePath
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}

		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}

		if err := gotenv.Load(path); err != nil {
			return err
		}
	}

	return nil
}

func applyFlagOverrides(cmd *cobra.Command) error {
	if cmd.Flags().Changed("api-url") {
		apiURL, err := cmd.Flags().GetString("api-url")
		if err != nil {
			return err
		}
		if strings.TrimSpace(apiURL) != "" {
			viper.Set("api_url", apiURL)
		}
	}

	return nil
}

func configPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".expenserc"), nil
}

func saveConfig() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	return viper.WriteConfigAs(path)
}

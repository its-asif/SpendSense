package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		Use:           "expense",
		Short:         "SpendSense CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig()
		},
	}

	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.expenserc)")
	cmd.PersistentFlags().String("api-url", "http://localhost:8080", "API server URL")

	_ = viper.BindPFlag("api_url", cmd.PersistentFlags().Lookup("api-url"))
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

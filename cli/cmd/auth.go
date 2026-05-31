package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	cmd.AddCommand(newAuthRegisterCmd())
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthLogoutAllCmd())
	cmd.AddCommand(newAuthMeCmd())
	cmd.AddCommand(newAuthRefreshCmd())

	return cmd
}

func newAuthRegisterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "register",
		Short: "Register a new account",
		RunE: func(cmd *cobra.Command, args []string) error {
			email := promptLine("Email")
			password := promptSecret("Password")
			confirm := promptSecret("Confirm Password")
			if password != confirm {
				return fmt.Errorf("passwords do not match")
			}

			client := newAPIClient()
			resp, err := client.Register(cmd.Context(), email, password)
			if err != nil {
				return err
			}

			return saveAuthSession(resp.AccessToken, resp.RefreshToken, resp.User.ID, resp.User.Email, resp.User.BaseCurrency, resp.User.Timezone, resp.User.Locale)
		},
	}
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login to your account",
		RunE: func(cmd *cobra.Command, args []string) error {
			email := promptLine("Email")
			password := promptSecret("Password")

			client := newAPIClient()
			resp, err := client.Login(cmd.Context(), email, password, "")
			if err != nil {
				// If server requires TOTP, prompt the user and retry
				if apiErr, ok := err.(*APIError); ok && apiErr.Code == "TOTP_REQUIRED" {
					for attempts := 0; attempts < 3; attempts++ {
						code := promptLine("Two-factor code")
						resp, err = client.Login(cmd.Context(), email, password, code)
						if err == nil {
							break
						}
						if apiErr, ok := err.(*APIError); ok && apiErr.Code == "INVALID_CODE" {
							fmt.Println("Invalid code, try again.")
							continue
						}
						return err
					}
					if err != nil {
						return err
					}
				} else {
					return err
				}
			}

			return saveAuthSession(resp.AccessToken, resp.RefreshToken, resp.User.ID, resp.User.Email, resp.User.BaseCurrency, resp.User.Timezone, resp.User.Locale)
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Logout from the current session",
		RunE: func(cmd *cobra.Command, args []string) error {
			refreshToken := strings.TrimSpace(viper.GetString("refresh_token"))
			if refreshToken == "" {
				return fmt.Errorf("no refresh token saved in config")
			}

			client := newAPIClient()
			if err := client.Logout(cmd.Context(), refreshToken); err != nil {
				return err
			}

			clearAuthSession()
			return saveConfig()
		},
	}
}

func newAuthLogoutAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout-all",
		Short: "Logout from all sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			if err := client.LogoutAll(cmd.Context()); err != nil {
				return err
			}

			clearAuthSession()
			return saveConfig()
		},
	}
}

func newAuthMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Show the current authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := newAPIClient()
			resp, err := client.Me(cmd.Context())
			if err != nil {
				return err
			}

			fmt.Printf("%s <%s>\n", resp.UserID, resp.Email)
			return nil
		},
	}
}

func newAuthRefreshCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "refresh",
		Short: "Refresh the access token",
		RunE: func(cmd *cobra.Command, args []string) error {
			email := strings.TrimSpace(viper.GetString("email"))
			refreshToken := strings.TrimSpace(viper.GetString("refresh_token"))
			if email == "" || refreshToken == "" {
				return fmt.Errorf("missing saved email or refresh token")
			}

			client := newAPIClient()
			accessToken, err := client.Refresh(cmd.Context(), email, refreshToken)
			if err != nil {
				return err
			}

			viper.Set("access_token", accessToken)
			return saveConfig()
		},
	}
}

func saveAuthSession(accessToken, refreshToken, userID, email, baseCurrency, timezone, locale string) error {
	viper.Set("access_token", accessToken)
	viper.Set("refresh_token", refreshToken)
	viper.Set("user_id", userID)
	viper.Set("email", email)
	if baseCurrency != "" {
		viper.Set("base_currency", baseCurrency)
	}
	if timezone != "" {
		viper.Set("timezone", timezone)
	}
	if locale != "" {
		viper.Set("locale", locale)
	}
	if viper.GetString("api_url") == "" {
		viper.Set("api_url", "http://localhost:8080")
	}

	if err := saveConfig(); err != nil {
		return err
	}

	fmt.Printf("Logged in as %s\n", email)
	return nil
}

func clearAuthSession() {
	viper.Set("access_token", "")
	viper.Set("refresh_token", "")
	viper.Set("user_id", "")
	viper.Set("email", "")
}

func promptLine(label string) string {
	fmt.Printf("%s: ", label)
	reader := bufio.NewReader(os.Stdin)
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value)
}

func promptSecret(label string) string {
	fmt.Printf("%s: ", label)
	password, _ := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	return strings.TrimSpace(string(password))
}

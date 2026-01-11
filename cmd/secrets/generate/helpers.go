package generate

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	internalIO "github.com/kloudkit/ws-cli/internals/io"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

type outputConfig struct {
	file  string
	mode  string
	force bool
	raw   bool
}

func getOutputConfig(cmd *cobra.Command) outputConfig {
	outputFile, _ := cmd.Flags().GetString("output")
	modeStr, _ := cmd.Flags().GetString("mode")
	force, _ := cmd.Flags().GetBool("force")
	raw, _ := cmd.Flags().GetBool("raw")

	return outputConfig{
		file:  outputFile,
		mode:  modeStr,
		force: force,
		raw:   raw,
	}
}

func generateMasterKey(cmd *cobra.Command, cfg outputConfig, keyLength int) error {
	if keyLength <= 0 {
		return errors.New("invalid key length")
	}

	key := make([]byte, keyLength)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	encodedKey := base64.StdEncoding.EncodeToString(key)

	if cfg.file == "" {
		if cfg.raw {
			fmt.Fprintln(cmd.OutOrStdout(), encodedKey)
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Header().Render("Master Key"))
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", styles.Code().Render(encodedKey))
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Muted().Render("💡 Store this key securely - you'll need it to encrypt/decrypt secrets"))
		return nil
	}

	if err := internalIO.WriteSecureFile(cfg.file, []byte(encodedKey+"\n"), cfg.mode, cfg.force); err != nil {
		return err
	}

	if !cfg.raw {
		styles.PrintSuccessWithDetailsCode(cmd.OutOrStdout(), "✓ Master key written to "+cfg.file, [][]string{
			{"Output", cfg.file},
		})
	}
	return nil
}

func generateLoginHash(cmd *cobra.Command, cfg outputConfig) error {
	password, err := internalIO.ReadPasswordFromReader(cmd.InOrStdin())
	if err != nil {
		return err
	}

	hash, err := internalSecrets.HashPasswordForWorkspace(password)
	if err != nil {
		return err
	}

	if cfg.file == "" {
		if cfg.raw {
			fmt.Fprintln(cmd.OutOrStdout(), hash)
			return nil
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Header().Render("Workspace Password Hash"))
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", styles.Code().Render(hash))
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", styles.Muted().Render("💡 Use this hash for WS_AUTH_PASSWORD_HASHED"))
		return nil
	}

	if err := internalIO.WriteSecureFile(cfg.file, []byte(hash+"\n"), cfg.mode, cfg.force); err != nil {
		return err
	}

	if !cfg.raw {
		styles.PrintSuccessWithDetailsCode(cmd.OutOrStdout(), "✓ Password hash written to "+cfg.file, [][]string{
			{"Output", cfg.file},
		})
	}
	return nil
}

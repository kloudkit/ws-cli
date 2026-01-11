package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a master key",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := getOutputConfig(cmd)
		keyLength, _ := cmd.Flags().GetInt("length")

		if keyLength <= 0 {
			return errors.New("invalid key length")
		}

		key := make([]byte, keyLength)
		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}

		encodedKey := base64.StdEncoding.EncodeToString(key)

		return handleCustomOutput(cmd, cfg, encodedKey, fmt.Sprintf("✓ Master key written to %s", cfg.file), func(out io.Writer) {
			fmt.Fprintf(out, "%s\n", styles.Header().Render("Master Key"))
			fmt.Fprintf(out, "  %s\n", styles.Code().Render(encodedKey))
			fmt.Fprintf(out, "%s\n", styles.Muted().Render("💡 Store this key securely - you'll need it to encrypt/decrypt secrets"))
		})
	},
}

func init() {
	generateCmd.Flags().String("output", "", "Output file (default stdout)")
	generateCmd.Flags().Int("length", 32, "Length in bytes")
}

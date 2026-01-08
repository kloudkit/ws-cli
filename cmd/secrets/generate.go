package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/kloudkit/ws-cli/internals/io"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a master key",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		keyLength, _ := cmd.Flags().GetInt("length")
		outputFile, _ := cmd.Flags().GetString("output")
		modeStr, _ := cmd.Flags().GetString("mode")
		force, _ := cmd.Flags().GetBool("force")
		raw, _ := cmd.Flags().GetBool("raw")

		if keyLength <= 0 {
			return errors.New("invalid key length")
		}

		key := make([]byte, keyLength)
		if _, err := rand.Read(key); err != nil {
			return fmt.Errorf("failed to generate key: %w", err)
		}

		encodedKey := base64.StdEncoding.EncodeToString(key)

		if outputFile == "" {
			if raw {
				fmt.Fprintln(cmd.OutOrStdout(), encodedKey)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), styles.Title().Render("Master key"))
				fmt.Fprintln(cmd.OutOrStdout(), styles.Code().Render(encodedKey))
			}
			return nil
		}

		if err := io.WriteSecureFile(outputFile, []byte(encodedKey+"\n"), modeStr, force); err != nil {
			return err
		}

		if !raw {
			fmt.Fprintln(cmd.OutOrStdout(), styles.Success().Render(fmt.Sprintf("Master key written to %s", outputFile)))
		}

		return nil
	},
}

func init() {
	generateCmd.Flags().String("output", "", "Output file (default stdout)")
	generateCmd.Flags().Int("length", 32, "Length in bytes")
}

package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/kloudkit/ws-cli/cmd/cli"
	internalIO "github.com/kloudkit/ws-cli/internals/io"
	internalSecrets "github.com/kloudkit/ws-cli/internals/secrets"
	"github.com/kloudkit/ws-cli/internals/styles"
	"github.com/spf13/cobra"
)

func transform(use, short, long, title, done string, newline bool, apply func(input string, key []byte) (string, error)) *cobra.Command {
	cmd := cli.New(use, "0.2.0", short, long, func(cmd *cobra.Command, args []string) error {
		key, err := internalSecrets.ResolveMasterKey(cli.String(cmd, "master"))
		if err != nil {
			return err
		}

		input, err := internalIO.ReadInput(args[0], cmd.InOrStdin())
		if err != nil {
			return err
		}

		value, err := apply(input, key)
		if err != nil {
			return err
		}

		return emit(cmd, value, done, newline, func(out io.Writer) {
			styles.PrintTitle(out, title)
			styles.PrintKeyValues(out, styles.Pair{Key: "Value", Value: value})
		})
	})
	cmd.Args = cobra.ExactArgs(1)

	return cmd
}

var encryptCmd = transform("encrypt <plaintext|->", "Encrypt a plaintext value",
	"Encrypt a value under the master key. Reads the plaintext from the argument or stdin (-); writes the ciphertext to stdout, or a file with --output.",
	"Encrypted Value", "Secret encrypted", true,
	func(input string, key []byte) (string, error) {
		encrypted, err := internalSecrets.Encrypt([]byte(strings.TrimSpace(input)), key)
		if err != nil {
			return "", fmt.Errorf("encryption failed: %w", err)
		}

		return encrypted, nil
	})

var decryptCmd = transform("decrypt <encrypted|->", "Decrypt an encrypted value",
	"Decrypt a value produced by encrypt, under the master key. Reads from the argument or stdin (-); writes the plaintext to stdout, or a file with --output.",
	"Decrypted Value", "Secret decrypted", false,
	func(input string, key []byte) (string, error) {
		decrypted, err := internalSecrets.Decrypt(internalSecrets.NormalizeEncrypted(input), key)
		return string(decrypted), err
	})

var masterCmd = cli.New("master", "0.2.0", "Generate a cryptographically secure master key",
	"Generate a random master key, printed base64-encoded, for encrypt, decrypt, and the seed engine. --length sets the byte size (default 32).",
	func(cmd *cobra.Command, args []string) error {
		length := cli.Int(cmd, "length")
		if length <= 0 {
			return errors.New("invalid key length")
		}

		key := make([]byte, length)
		rand.Read(key)

		return emitKey(cmd, base64.StdEncoding.EncodeToString(key), "Master key written", "Master Key",
			"Store this key securely. You need it to encrypt and decrypt secrets.")
	})

var loginCmd = cli.New("login", "0.2.0", "Generate a workspace password hash for authentication",
	"Prompt for a password and print its hash for the workspace server login (WS_AUTH_PASSWORD_HASHED). Store the hash, never the password.",
	func(cmd *cobra.Command, args []string) error {
		password, err := internalIO.ReadPasswordFromReader(cmd.InOrStdin())
		if err != nil {
			return err
		}

		hash, err := internalSecrets.HashPasswordForWorkspace(password)
		if err != nil {
			return err
		}

		return emitKey(cmd, hash, "Password hash written", "Workspace Password Hash", "Use this hash for WS_AUTH_PASSWORD_HASHED.")
	})

var materializeCmd = cli.New("materialize", "0.5.0", "Project the configured master key to its conventional secret path",
	"Persist WS_SECRETS_MASTER_KEY to /run/secrets/workspace/secrets/master_key so the key outlives the editor's environment scrub. A no-op when the key is unset or the path already holds one.",
	func(cmd *cobra.Command, args []string) error {
		path, err := internalSecrets.MaterializeMasterKey()
		if err != nil {
			return err
		}

		if path != "" {
			styles.PrintSuccess(cmd.OutOrStdout(), "Master key projected to "+path)
		}

		return nil
	})

var generateCmd = cli.Group("generate", "0.2.0", "Generate master keys or login password hashes",
	"Generate the credentials the workspace needs: a master key for secrets, or a login password hash for the server.", "",
	masterCmd, loginCmd)

var SecretsCmd = cli.Group("secrets", "0.2.0", "Manage encryption and decryption of secrets",
	"Encrypt and decrypt values under a master key, and generate the keys themselves. Encrypted values are what the seed engine's secrets: map stores and decrypts at boot.",
	`# Generate a master key
ws-cli secrets generate master

# Encrypt a value under it
ws-cli secrets encrypt "s3cr3t" --master ~/.ws/master.key`,
	encryptCmd, decryptCmd, generateCmd, materializeCmd)

func emitKey(cmd *cobra.Command, value, done, title, hint string) error {
	return emit(cmd, value, done, true, func(out io.Writer) {
		styles.PrintTitle(out, title)
		styles.PrintText(out, value)
		fmt.Fprintln(out)
		styles.PrintMuted(out, hint)
	})
}

func emit(cmd *cobra.Command, value, done string, newline bool, styled func(io.Writer)) error {
	out := cmd.OutOrStdout()
	file, raw := cli.String(cmd, "output"), cli.Bool(cmd, "raw")

	switch {
	case file != "":
		if err := internalIO.WriteSecureFile(file, []byte(value+"\n"), cli.String(cmd, "mode"), cli.Bool(cmd, "force")); err != nil {
			return err
		}

		if !raw {
			styles.PrintSuccess(out, done, styles.Pair{Key: "Output", Value: file})
		}
	case raw && newline:
		fmt.Fprintln(out, value)
	case raw:
		fmt.Fprint(out, value)
	default:
		styled(out)
	}

	return nil
}

func init() {
	SecretsCmd.PersistentFlags().String("master", "", "Master key or path to key file")
	SecretsCmd.PersistentFlags().String("output", "", "Write output to file instead of stdout")
	SecretsCmd.PersistentFlags().String("mode", "", "File permissions (e.g., 0o600, 384), only when --output is used")
	SecretsCmd.PersistentFlags().Bool("force", false, "Overwrite existing files")
	SecretsCmd.PersistentFlags().Bool("raw", false, "Output without styling")
	masterCmd.Flags().Int("length", 32, "Key length in bytes")

	for _, cmd := range []*cobra.Command{masterCmd, loginCmd, materializeCmd} {
		cmd.Args = cobra.NoArgs
	}
}

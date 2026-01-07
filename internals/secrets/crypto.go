package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/crypto/argon2"
	"gopkg.in/yaml.v3"
)

const (
	Argon2Time    = 3
	Argon2Memory  = 64 * 1024 // 64MB
	Argon2Threads = 4
	Argon2KeyLen  = 32
	SaltLen       = 16
	NonceLen      = 12
)

func Encrypt(plainText []byte, masterKey []byte) (string, error) {
	salt := make([]byte, SaltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	aesGCM, err := deriveKeyAndGCM(masterKey, salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	cipherText := aesGCM.Seal(nonce, nonce, plainText, nil)
	return fmt.Sprintf("argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		Argon2Memory, Argon2Time, Argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(cipherText)), nil
}

func Decrypt(encodedValue string, masterKey []byte) ([]byte, error) {
	parts := strings.Split(encodedValue, "$")
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid encoded format")
	}

	if parts[0] != "argon2id" {
		return nil, fmt.Errorf("unsupported algorithm: %s", parts[0])
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return nil, fmt.Errorf("failed to decode salt: %w", err)
	}

	cipherTextWithNonce, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	aesGCM, err := deriveKeyAndGCM(masterKey, salt, Argon2Time, Argon2Memory, Argon2Threads, Argon2KeyLen)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(cipherTextWithNonce) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	return aesGCM.Open(nil, cipherTextWithNonce[:nonceSize], cipherTextWithNonce[nonceSize:], nil)
}

func deriveKeyAndGCM(masterKey, salt []byte, time, memory uint32, threads uint8, keyLen uint32) (cipher.AEAD, error) {
	key := argon2.IDKey(masterKey, salt, time, memory, threads, keyLen)
	defer zeroBytes(key)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return cipher.NewGCM(block)
}

func zeroBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}

func LoadVaultFromFile(filePath string) (*Vault, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault file: %w", err)
	}

	var vault Vault
	if err := yaml.Unmarshal(data, &vault); err != nil {
		return nil, fmt.Errorf("failed to parse vault YAML: %w", err)
	}

	return &vault, nil
}

func (v *Vault) EncryptAll(masterKey []byte) error {
	for i := range v.Secrets {
		secret := &v.Secrets[i]

		plaintext, err := secret.ReadPlaintextValue()
		if err != nil {
			return fmt.Errorf("failed to read secret value for %s: %w", secret.Destination, err)
		}

		encrypted, err := Encrypt(plaintext, masterKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt secret for %s: %w", secret.Destination, err)
		}

		secret.Value = encrypted
	}

	return nil
}

func (v *Vault) DecryptAll(masterKey []byte, opts WriteOptions) error {
	for i := range v.Secrets {
		secret := &v.Secrets[i]

		if secret.Value == "" {
			return fmt.Errorf("secret for %s has empty value", secret.Destination)
		}

		decrypted, err := Decrypt(secret.Value, masterKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt secret for %s: %w", secret.Destination, err)
		}

		effectiveForce := opts.Force || secret.Force

		writeOpts := WriteOptions{
			Force:  effectiveForce,
			DryRun: opts.DryRun,
		}

		if err := WriteSecret(secret, decrypted, writeOpts); err != nil {
			return fmt.Errorf("failed to write secret for %s: %w", secret.Destination, err)
		}
	}

	return nil
}

func (v *Vault) ToYAML() ([]byte, error) {
	return yaml.Marshal(v)
}

func (v *Vault) AddSecret(value, dest, secretType string, force bool) {
	v.Secrets = append(v.Secrets, Secret{
		Type:        secretType,
		Value:       value,
		Destination: dest,
		Force:       force,
	})
}

func (v *Vault) SaveToFile(path string) error {
	yamlData, err := v.ToYAML()
	if err != nil {
		return fmt.Errorf("failed to marshal vault: %w", err)
	}

	if err := os.WriteFile(path, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write vault file: %w", err)
	}

	return nil
}

func EncryptToVault(value []byte, vaultPath, dest, secretType string, masterKey []byte, force, dryRun bool) error {
	var vault *Vault

	if _, err := os.Stat(vaultPath); err == nil {
		loadedVault, err := LoadVaultFromFile(vaultPath)
		if err != nil {
			return err
		}
		vault = loadedVault
	} else {
		vault = &Vault{Secrets: []Secret{}}
	}

	encrypted, err := Encrypt(value, masterKey)
	if err != nil {
		return fmt.Errorf("encryption failed: %w", err)
	}

	vault.AddSecret(encrypted, dest, secretType, force)

	if dryRun {
		return nil
	}

	return vault.SaveToFile(vaultPath)
}

func DecryptVault(vaultPath string, masterKey []byte, force, dryRun bool) error {
	vault, err := LoadVaultFromFile(vaultPath)
	if err != nil {
		return err
	}

	opts := WriteOptions{
		Force:  force,
		DryRun: dryRun,
	}

	return vault.DecryptAll(masterKey, opts)
}

func DecryptSingle(encrypted, dest string, masterKey []byte, force, dryRun bool) ([]byte, error) {
	decrypted, err := Decrypt(encrypted, masterKey)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	if dest == "" || dest == "stdout" {
		return decrypted, nil
	}

	secret := &Secret{Destination: dest}
	opts := WriteOptions{Force: force, DryRun: dryRun}

	if err := WriteSecret(secret, decrypted, opts); err != nil {
		return nil, err
	}

	return decrypted, nil
}

# WS-CLI Secrets Subcommand – Final Implementation Plan

---

## 1. CLI Structure

Base command:

```bash
ws-cli secrets
```

---

## 2. Subcommands

### 2.1 Encrypt a Secret

```bash
ws-cli secrets encrypt \
  [--value <value>] \
  [--type <type>] \
  [--dest <file|env>] \
  [--vault <vault.yaml>] \
  [--master <key|path>] \
  [--force] \
  [--dry-run]
```

**Behavior**

* Encrypts a single secret
* Writes encrypted output either:

  * directly to a vault file, or
  * prints encrypted value (if no vault is provided)

---

### 2.2 Decrypt Secrets

```bash
ws-cli secrets decrypt \
  [--encrypted <value>] \
  [--dest <file|env|stdout>] \
  [--vault <vault.yaml>] \
  [--master <key|path>] \
  [--force] \
  [--dry-run]
```

**Behavior**

* Decrypts either:

  * a single encrypted value, or
  * all secrets in a vault
* Output destinations:

  * file → write to disk
  * env → append to shell environment file
  * stdout → print only

---

### 2.3 Create an Encrypted Vault

```bash
ws-cli secrets vault \
  --input plain.yaml \
  [--output <file>] \
  [--master <key|path>] \
  [--force] \
  [--dry-run]
```

**Behavior**

* Reads plaintext secrets from files or environment variables
* Encrypts them into a portable vault YAML or outputs to stdout

---

### 2.4 Generate a Master Key

```bash
ws-cli secrets generate \
  [--output <file>] \
  [--length 32] \
  [--force]
```

**Behavior**

* Generates cryptographically secure random bytes
* Default length: **32 bytes (256-bit)**
* Output is Base64-encoded
* Used as the master key for all encryption/decryption

---

## 3. Global Flags & Semantics

| Flag           | Description                                   |
| -------------- | --------------------------------------------- |
| `--master` | Literal key or path to key file               |
| `--force`   | Global overwrite flag (takes precedence)      |
| `--dry-run`    | Perform full decrypt/encrypt but do not write |

---

## 4. Master Key Resolution

Resolution order:

1. `--master`
2. `WS_SECRETS_MASTER_KEY`
3. `WS_SECRETS_MASTER_KEY_FILE` (defaults to `/etc/workspace/master.key`)
4. Error if not found

**Key interpretation rule**

* If argument points to an existing file → read file contents
* Otherwise → treat as literal key

The master key is **never logged**.

---

## 5. Vault File Format

```yaml
secrets:
  - type: kubeconfig
    value: <encrypted>
    destination: /home/dev/.kube/config
    force: true
  - type: ssh
    value: <encrypted>
    destination: ~/.ssh/id_rsa
  - type: env
    value: <encrypted>
    destination: MY_SECRET_ENV
```

### Rules

* `destination` may be:

  * file path
  * environment variable name
* `force` is optional and applies per-secret
* CLI `--force` **overrides** YAML `force`

---

## 6. Destination Expansion & Validation

### Expansion (performed first)

* `~`
* `$HOME`
* `$VAR`

### Validation

* **File destinations**

  * Must match approved path prefixes
  * Validated after expansion and normalization

```go
var allowedPaths = []string{
  "/home/dev/.kube/",
  "/home/dev/.ssh/",
  "/etc/secrets/",
}
```

* **Environment destinations**

  * Skip path whitelist
  * Must match valid env name regex:

    ```text
    ^[A-Z_][A-Z0-9_]*$
    ```

Invalid destinations cause failure or skip.

---

## 7. Encryption & Key Derivation

### Encryption

* AES-256-GCM
* Output encoded as Base64

### Key Derivation

* **Argon2id only**
* Fixed parameters (vault-portable):

```text
time=3
memory=64MB
threads=4
keyLen=32
```

### Encoding Format

```text
argon2id$v=19$m=65536,t=3,p=4$<salt>$<ciphertext>
```

Salt is generated per secret and stored with ciphertext.

---

## 8. Secret Data Model

```go
type Secret struct {
  Type        string `yaml:"type"`
  Value       string `yaml:"value"`
  Destination string `yaml:"destination"`
  Force       bool   `yaml:"force,omitempty"`
}
```

---

## 9. Type-Based File Modes

```go
var typeFileModes = map[string]os.FileMode{
  "kubeconfig": 0600,
  "ssh":        0600,
  "password":   0600,
  "config":     0644,
}
```

### Rules

* Applied **only to file destinations**
* Ignored for:

  * environment variables
  * stdout output

---

## 10. Vault Creation Flow

**Input YAML (plaintext):**

```yaml
secrets:
  - type: kubeconfig
    destination: /home/dev/.kube/config
  - type: env
    destination: MY_SECRET
```

### Steps

1. Load plaintext YAML
2. For each secret:

   * Expand destination
   * Validate destination
   * Read value from file or environment
   * Encrypt using AES-GCM with master key
   * Store encrypted value in memory
3. Output
  * If `--stdout` → print entire encrypted vault YAML to stdout
  * Else → write to --output file
4. Respect `--force`, `--dry-run`

---

## 11. Decryption Flow

### Output Rules

| Destination | Action                |
| ----------- | --------------------- |
| File path   | Write file            |
| Env         | Append to `~/.zshenv` |
| Stdout      | Print decrypted value |

### Environment Handling

* Always use `~/.zshenv`
* Only append
* Do **not** overwrite existing entries
* If variable already exists, skip

### Steps

1. Load vault or encrypted value
2. Decrypt secret
3. Validate destination
4. Apply effective force:

   ```go
   effectiveForce := cliForce || secret.Force
   ```
5. Write output (unless dry-run)
6. Apply file mode if applicable

---

## 12. Dry-Run Behavior

* Full encryption/decryption occurs
* **No writes**
* Outputs exactly what *would* be written:

  * file path + permissions
  * `export VAR=...`
  * stdout values

⚠️ Dry-run intentionally reveals secrets.

---

## 13. Security Considerations

* Never log:

  * master key
  * decrypted values (unless stdout/dry-run)
* Zero decrypted byte buffers where possible
* Encrypted values stored as strings only
* Fail-fast on invalid YAML or unreadable destinations

---

## 14. Validation & Error Handling

* Validate:

  * YAML structure
  * destination safety
  * file write permissions
* Skip or abort behavior must be explicit

# Overview

Add a `ws-cli files sync` command that syncs files/directories from a YAML manifest using native Go file operations. This consolidates and simplifies the two original plan attempts.

## Scope

**In scope:** ws-cli implementation only
**Out of scope:** Workspace repo changes (startup scripts, env.reference.yaml, integration tests)

## Simplifications from Original Plan

1. **No Ansible at all** - Use native Go file operations via existing `internals/io` module. This avoids Jinja2 templating security concerns entirely and keeps the implementation simple.

2. **No templating** - Content is written literally. No Jinja2, no variable substitution in file contents.

3. **Allow `~` in paths** - Expand `~` to home directory before validation. Paths are validated after expansion.

4. **Reuse existing internals** - Use `internals/io.CopyFile()`, `io.WriteSecureFile()`, and `os.MkdirAll()` for all file operations. Add validation functions to existing `internals/path/support.go`.

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `internals/path/security.go` | Create | Denylist definitions and validation logic |
| `internals/path/security_test.go` | Create | Security validation tests |
| `internals/path/support.go` | Modify | Add `ValidateDestination()`, `ValidateSource()` wrappers |
| `internals/config/defaults.go` | Modify | Add `EnvStartupFilesSync` constant |
| `internals/files/manifest.go` | Create | YAML parsing and validation |
| `internals/files/sync.go` | Create | Sync orchestration using native Go |
| `cmd/files/files.go` | Create | Parent command |
| `cmd/files/sync.go` | Create | Sync subcommand |
| `cmd/secrets/vault.go` | Modify | Add `path.ValidateDestination()` call |
| `cmd/root.go` | Modify | Register `files.FilesCmd` |

## Manifest Schema

- Paths: absolute or `~` (expanded to home directory)
- Content: written literally (no templating, no variable expansion)

```yaml
files:
  - copy:
      src: /workspace/configs/app.conf
      dest: ~/.config/app/config.conf
      mode: "0644"

  - copy:
      src: /run/secrets/gitconfig
      dest:
        - ~/.gitconfig
        - /workspace/.gitconfig
      mode: "0600"

  - content:
      data: |
        DEBUG=false
        LOG_LEVEL=info
      dest: ~/.env
      mode: "0600"

  - ensure:
      path: ~/.local/logs
      state: directory
      mode: "0755"
```

## Data Structures

```go
// internals/files/manifest.go

type SyncManifest struct {
    Files []SyncFile `yaml:"files"`
}

type SyncFile struct {
    Copy    *CopyOp    `yaml:"copy,omitempty"`
    Content *ContentOp `yaml:"content,omitempty"`
    Ensure  *EnsureOp  `yaml:"ensure,omitempty"`
}

type CopyOp struct {
    Src  string      `yaml:"src"`
    Dest StringOrList `yaml:"dest"`
    Mode string      `yaml:"mode,omitempty"`
}

type ContentOp struct {
    Data   string `yaml:"data,omitempty"`
    Base64 string `yaml:"data_base64,omitempty"`
    Dest   string `yaml:"dest"`
    Mode   string `yaml:"mode,omitempty"`
}

type EnsureOp struct {
    Path  string `yaml:"path"`
    State string `yaml:"state"` // directory, file
    Mode  string `yaml:"mode,omitempty"`
}

type StringOrList []string // Custom unmarshaler for single string or list
```

## Security Model

### Threat Model

Users could attempt to compromise the workspace by writing to sensitive locations:

1. **Autoload script injection** - Write malicious scripts to autoload directories that execute during workspace startup
2. **SSH key injection** - Write to `~/.ssh/authorized_keys` to grant unauthorized access

**Note:** System paths like `/etc/`, `/usr/`, `/var/` are already protected by Linux file permissions since the container runs as non-root user `kloud`. This was verified by integration test `test_vault_blocked_system_path` which confirms writes to `/etc/sudoers.d/` fail with "permission denied".

### Defense Strategy: Denylist for User-Writable Paths

Since system paths are protected by OS permissions, we only need application-level protection for user-writable sensitive paths within the allowed directories.

## Path Validation (in existing support.go)

**Location:** `internals/path/support.go`

Add validation functions to existing module:

```go
func ValidateDestination(p string) error {
    // 1. Expand ~ to home directory
    // 2. Clean with filepath.Clean()
    // 3. Require absolute path
    // 4. Check against allowed prefixes
    // 5. Check against denied paths/patterns
    // 6. Resolve symlinks and re-validate target
}

func ValidateSource(p string) error {
    // Same logic, different allowed/denied lists
}
```

### Allowed Prefixes (Allowlist)

**Destinations:**

- `$HOME` (user's home directory)
- `/workspace`
- `/tmp`

Note: System paths (`/etc/`, `/usr/`, `/var/`, etc.) are inherently blocked because they're not in the allowlist AND Linux file permissions prevent writes from the non-root `kloud` user.

**Sources (additional):**

- `/run/secrets`

### Denied Paths (Denylist)

Only user-writable sensitive paths need application-level blocking. System paths (`/etc/`, `/usr/`, `/var/`, etc.) are already protected by Linux file permissions.

**Workspace protected paths:**

- `/workspace/.kloudkit/` - Workspace internal configuration
- `/workspace/.autoload/` - Startup scripts executed during boot
- `/workspace/.startup/` - Alternative startup script location
- `/workspace/.hooks/` - Lifecycle hooks
- `/workspace/.devcontainer/` - Dev container configuration

**Home directory protected paths:**

- `~/.ssh/authorized_keys` - SSH access control
- `~/.ssh/authorized_keys2` - SSH access control (alternate)
- `~/.ssh/rc` - SSH login script
- `~/.ssh/environment` - SSH environment
- `~/.gnupg/` - GPG keys and config
- `~/.kloudkit/` - CLI internal config

**Pattern-based denials (substring match):**

- Paths containing `autoload` (any case)
- Paths containing `.startup`

### Symlink Protection

Validate both the requested path AND the resolved path after symlink resolution:

```go
func ValidateDestination(p string) error {
    expanded, err := Expand(p)
    if err != nil {
        return err
    }

    // Validate the literal path first
    if err := validateAgainstRules(expanded, destAllowed, destDenied); err != nil {
        return err
    }

    // If path exists, resolve symlinks and re-validate
    if resolved, err := filepath.EvalSymlinks(expanded); err == nil && resolved != expanded {
        if err := validateAgainstRules(resolved, destAllowed, destDenied); err != nil {
            return fmt.Errorf("symlink target blocked: %w", err)
        }
    }

    return nil
}
```

This prevents attacks where a user creates a symlink like `/workspace/innocent` → `/etc/passwd`.

### Validation Flow

```
1. Expand ~ to home directory using path.Expand()
2. Clean with filepath.Clean() to normalize ..
3. Require absolute path (starts with /)
4. Check against allowed prefixes (must match at least one)
5. Check against denied paths (must not match any)
6. Check against denied patterns (must not contain any)
7. If path exists, resolve symlinks and repeat steps 4-6 on target
```

### Error Messages

Clear, actionable error messages help legitimate users:

```
path '/workspace/.autoload/script.sh' is protected: startup scripts cannot be modified
path '/etc/passwd' is outside allowed directories (allowed: $HOME, /workspace, /tmp)
path '/workspace/link' resolves to '/etc/shadow' which is protected
```

### Extensibility

The denylist is defined as package-level variables, making it easy to extend:

```go
// internals/path/security.go

// System paths (/etc/, /usr/, /var/, etc.) are NOT included here
// because Linux file permissions already block writes from the
// non-root 'kloud' user. Only user-writable sensitive paths need
// application-level protection.

var DeniedDestSuffixes = []string{
    "/.kloudkit/",
    "/.autoload/",
    "/.startup/",
    "/.hooks/",
    "/.devcontainer/",
    "/.gnupg/",
}

var DeniedDestExact = []string{
    "/.ssh/authorized_keys",
    "/.ssh/authorized_keys2",
    "/.ssh/rc",
    "/.ssh/environment",
}

var DeniedDestPatterns = []string{
    "autoload",
    ".startup",
}
```

To add additional protection, append to the relevant slice. No code changes needed beyond updating the lists.

### Future Considerations

If users need legitimate access to protected paths (e.g., custom SSH config), consider:

1. **Explicit opt-in flag**: `--allow-protected` with confirmation prompt
2. **Separate allowlist file**: Admin-managed list of exceptions
3. **Per-path override**: `force: true` in manifest with warning output

These are not implemented in this plan but the architecture supports adding them later.

## Vault Security Fix

**Current state:** Linux file permissions already protect system paths (`/etc/`, `/usr/`, etc.) since the container runs as non-root user `kloud`. This is verified by integration test `test_vault_blocked_system_path`.

**Remaining gap:** User-writable sensitive paths are not protected by OS permissions.

**Attack vectors to close:**

- Writing to `/workspace/.autoload/` to inject startup scripts
- Writing to `~/.ssh/authorized_keys` to grant SSH access
- Creating symlinks that point to protected paths

**Fix:** Add `path.ValidateDestination()` call in vault processing before writing files.

```go
// In secrets vault processing, before writing:
if err := path.ValidateDestination(secret.Path); err != nil {
    return fmt.Errorf("secret '%s': %w", secret.Name, err)
}
```

The same denylist rules apply to vault as to file sync, providing consistent security across both features.

## Native Go File Operations

Instead of Ansible, use existing `internals/io` functions:

| Operation | Go Implementation |
|-----------|-------------------|
| `copy` | `io.CopyFile()` + `os.Chmod()` |
| `content` | `io.WriteSecureFile()` |
| `ensure` (directory) | `os.MkdirAll()` + `os.Chmod()` |
| `ensure` (file) | `os.OpenFile()` + close (touch) |

This keeps the implementation simple and avoids templating security concerns.

## Command Interface

```
ws-cli files sync [--input=<path>]

Flags:
  --input  Path to YAML manifest (default: $WS_STARTUP_FILES_SYNC)
```

**Resolution order:**
1. `--input` flag if provided
2. `WS_STARTUP_FILES_SYNC` env var
3. Error if neither set

## Execution Flow

```
1. Parse --input flag or read WS_STARTUP_FILES_SYNC
2. Validate manifest path exists
3. Parse YAML manifest
4. Validate each file entry:
   - Exactly one operation type (copy/content/ensure)
   - Expand ~ and validate all paths
5. Execute operations using native Go:
   - copy: io.CopyFile() + os.Chmod()
   - content: io.WriteSecureFile()
   - ensure: os.MkdirAll() or touch
6. Print success/error summary
```

## Error Handling

- **Missing manifest path:** Error with usage hint
- **Invalid YAML:** Error with parse details
- **Path validation failure:** Error with specific file index and path
- **File operation failure:** Print error, continue with remaining files, exit 1 at end if any failed

## Implementation Order

**Phase 1: Security foundation (closes existing vulnerability)**
1. `internals/path/security.go` - Denylist definitions and core validation
2. `internals/path/security_test.go` - Comprehensive security tests
3. `internals/path/support.go` - Add `ValidateDestination()`, `ValidateSource()` wrappers
4. `cmd/secrets/vault.go` - Add validation call (immediately closes security gap)

**Phase 2: File sync feature**
5. `internals/config/defaults.go` - Add `EnvStartupFilesSync` constant
6. `internals/files/manifest.go` - Types + YAML parsing
7. `internals/files/sync.go` - Orchestration using native Go (reuses security validation)
8. `cmd/files/files.go` - Parent command
9. `cmd/files/sync.go` - Subcommand
10. `cmd/root.go` - Register `files.FilesCmd`

## Verification

**Path validation tests:**
```bash
go test ./internals/path/... -v
```

**Files sync - functional tests:**
1. Create test manifest at `/tmp/test-sync.yaml`
2. Run `ws-cli files sync --input=/tmp/test-sync.yaml`
3. Verify files created with correct permissions
4. Test `~` expansion works correctly

**Security tests - protected paths:**

Note: System paths (`/etc/`, `/usr/`, etc.) are protected by Linux file permissions.
This is verified by integration test `test_vault_blocked_system_path` in the workspace repo.

| Test Case | Path | Expected |
|-----------|------|----------|
| Autoload injection | `/workspace/.autoload/evil.sh` | Rejected: startup scripts protected |
| Autoload pattern | `/workspace/foo/autoload/bar` | Rejected: contains 'autoload' |
| SSH keys | `~/.ssh/authorized_keys` | Rejected: SSH access control protected |
| SSH rc | `~/.ssh/rc` | Rejected: SSH login script protected |
| Hooks | `/workspace/.hooks/pre-start` | Rejected: lifecycle hooks protected |
| Devcontainer | `/workspace/.devcontainer/devcontainer.json` | Rejected: dev container config protected |

**Security tests - symlink attacks:**

| Test Case | Setup | Expected |
|-----------|-------|----------|
| Symlink to /etc | Create `/workspace/link` → `/etc/passwd` | Rejected: symlink target blocked |
| Symlink to autoload | Create `/tmp/link` → `/workspace/.autoload/` | Rejected: symlink target blocked |
| Nested symlink | `/workspace/a` → `/workspace/b` → `/etc/` | Rejected: final target blocked |

**Security tests - allowed paths (should succeed):**

| Test Case | Path | Expected |
|-----------|------|----------|
| Home config | `~/.config/app/settings.json` | Allowed |
| Workspace file | `/workspace/myproject/.env` | Allowed |
| Tmp file | `/tmp/cache.txt` | Allowed |
| Home env | `~/.env` | Allowed |

**Vault security:**

System paths are already protected by Linux permissions (verified by `test_vault_blocked_system_path`).

1. Create vault with path to `/workspace/.autoload/` → Rejected
2. Create vault with path to `~/.ssh/authorized_keys` → Rejected
3. Create vault with path to `~/.env` → Allowed
4. Create vault with path to `/workspace/.env` → Allowed

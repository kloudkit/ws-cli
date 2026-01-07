# WS-CLI Secrets Implementation Tasks

## Completed ✅

### Phase 1: Foundation
- [x] Core encryption/decryption (AES-256-GCM with Argon2id)
- [x] Master key resolution (--master, env vars, default path)
- [x] Generate command (fully functional)
- [x] Data models (`Secret`, `Vault` structs with YAML tags)
- [x] Destination validation (path expansion, whitelist, env var regex)
- [x] File mode mapping by secret type
- [x] Tests for models and validation

### Phase 2: Core Features
- [x] File writer with proper permissions and force handling
- [x] Environment variable writer (~/.zshenv append with duplicate checking)
- [x] Vault command (encrypt plaintext vault to encrypted vault)
- [x] Enhanced decrypt command (single value + full vault decryption)
- [x] Enhanced encrypt command (single value + add to vault)
- [x] Secret value reading from files/env vars

### Phase 3: Enhancements
- [x] Dry-run support across all commands
- [x] Force flag handling (CLI overrides YAML)
- [x] Vault operations (load, encrypt all, decrypt all, to YAML)

---

## Remaining Tasks

### Testing & Quality

**Priority: Medium** - Ensure reliability

- [ ] Add tests for writer operations
  - File writing with permissions
  - Env var writing to ~/.zshenv
  - Duplicate checking
- [ ] Add tests for vault operations
  - LoadVaultFromFile
  - EncryptAll / DecryptAll
  - ToYAML
- [ ] Add integration tests for full workflows
  - End-to-end vault creation and decryption
  - Multiple secret types
  - Force and dry-run scenarios

**Files to create/modify:**
- `internals/secrets/writer_test.go` (new)
- `internals/secrets/crypto_test.go` (extend with vault ops tests)
- `cmd/secrets/secrets_test.go`

---

## Implementation Status

### ✅ Phase 1: Foundation (Complete)
- Data models & YAML support
- Destination validation
- Comprehensive tests

### ✅ Phase 2: Core Features (Complete)
- File operations
- Vault command implementation
- Decrypt command enhancements

### ✅ Phase 3: Enhancements (Complete)
- Environment variable operations
- Encrypt command vault updating
- Dry-run implementation

### 🔄 Phase 4: Quality (In Progress)
- Testing (basic tests complete, need more coverage)
- Error handling (implemented)
- Documentation (TODO)

---

## Files Created/Modified

### Created
- `internals/secrets/models.go` - Data structures, validation, file modes
- `internals/secrets/models_test.go` - Tests for models and validation
- `internals/secrets/writer.go` - File and env var writing

### Modified
- `internals/secrets/crypto.go` - Encryption/decryption primitives + vault operations
- `cmd/secrets/vault.go` - Full vault creation implementation
- `cmd/secrets/decrypt.go` - Single value + vault decryption
- `cmd/secrets/encrypt.go` - Single value + add to vault

### Existing (Unchanged)
- `internals/secrets/key.go` - Master key resolution
- `cmd/secrets/generate.go` - Master key generation

---

## Notes

- All core functionality is implemented and working
- Master key resolution supports --master flag, env vars, and default path
- Dry-run mode works across all commands
- Force flag correctly overrides YAML-level force settings
- Security validations enforce allowed path whitelist
- Type-based file permissions automatically applied

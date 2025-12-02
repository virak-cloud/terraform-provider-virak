# CI Workflow Implementation: Provider & Examples Validation

## Overview

The GitHub Actions workflow in `.github/workflows/examples-validate.yml` implements a comprehensive CI pipeline for the Terraform provider and its example modules.

## Workflow Triggers

- **Push to master branch**: Automatically runs on commits to the default branch
- **Workflow dispatch**: Allows manual reruns via GitHub UI

## Job Structure

### Job 1: `provider` (Build & Test Provider)
**Runs on**: `ubuntu-latest`

**Steps**:
1. **Checkout code** - Fetches repository
2. **Set up Go 1.24.x** - Configures Go environment with automated caching of `~/.cache/go-build` keyed by `**/go.sum`
3. **Build provider binary** - Compiles to `terraform-provider-virakcloud`
4. **Run provider tests** - Executes `go test -v ./...`
5. **Upload provider binary** - Persists artifact for the next job (5-day retention)

### Job 2: `examples` (Matrix Validation)
**Runs on**: `ubuntu-latest` (parallel matrix over 11 example modules)
**Depends on**: `provider` job

**Matrix examples**:
- 01_bucket through 11_advanced_volumes

**Per-example steps**:
1. **Checkout code** - Fetches repository
2. **Set up Terraform** - Installs latest Terraform version
3. **Download provider binary** - Retrieves artifact from provider job
4. **Make executable** - Sets chmod +x on the binary
5. **Create dev override config** - Generates `~/.terraformrc` with dev override pointing to workspace:
   ```hcl
   provider_installation {
     dev_overrides {
       "terraform.local/local/virakcloud" = "${{ github.workspace }}"
     }
     direct {}
   }
   ```
6. **Set environment variables**:
   - `TF_CLI_CONFIG_FILE=$HOME/.terraformrc`
   - `TF_VAR_virakcloud_token=fake_token_for_validation`
   - `TF_VAR_ssh_key_id=dummy_ssh_key`
   - `TF_LOG=DEBUG` (captures detailed logs)
   - `TF_LOG_PATH` (per-example log file)

7. **Terraform init** - Runs `terraform init -backend=false` (skips remote backend)
8. **Terraform validate** - Validates configuration syntax and structure
9. **Terraform fmt check** - Optional formatting validation (continues on error)
10. **Collect logs and state** - Archives terraform logs, `.terraform` directory, and `terraform.lock.hcl`
11. **Upload per-example logs** - Stores artifacts per example (7-day retention)

### Job 3: `summary` (Consolidation & Reporting)
**Runs on**: `ubuntu-latest`
**Depends on**: `examples` job
**Runs**: Always (even on failure)

**Steps**:
1. Downloads all artifacts from prior jobs
2. Lists validation results
3. Creates consolidated tar.gz archive of all logs
4. Uploads consolidated archive (7-day retention)

## Key Features

### Environment Setup
- **Dummy credentials**: Validation uses fake tokens/IDs to avoid real API calls
- **Backend disabled**: `-backend=false` flag prevents Terraform state backend initialization
- **Dev overrides**: Uses Terraform's local provider override mechanism instead of registry

### Logging & Debugging
- **Per-example logs**: `terraform-<example>.log` files captured via `TF_LOG_PATH`
- **State snapshots**: `.terraform` directories and lock files preserved
- **Consolidated archive**: All artifacts bundled for easy download
- **Debug output**: `TF_LOG=DEBUG` enabled for detailed troubleshooting

### Failure Handling
- **fail-fast: false** - Continues validating all examples even if one fails
- **continue-on-error** - Formatting checks don't block the job
- **artifacts on failure** - Logs uploaded regardless of outcome

## Answers to Design Considerations

### 1. Terraform Linting Approach
**Implemented**: Both `terraform fmt -check` (optional) + `terraform validate` (required)
- `terraform validate` is the primary check (enforced)
- `terraform fmt -check` runs as an optional step (`continue-on-error: true`)
- Provides code quality feedback without blocking builds

### 2. Artifact Scope
**Implemented**: Hybrid approach
- **Per-example uploads** - `example-logs-<name>` artifacts for quick investigation
- **Consolidated archive** - `validation-logs-consolidated` for complete audit trail
- Each example has 7-day retention, consolidated archive also has 7 days
- Balances accessibility (per-example) with completeness (consolidated)

## Running Examples Locally

To mirror the CI behavior locally:

```bash
# Build the provider
go build -o terraform-provider-virakcloud ./

# Create dev override config
cat > ~/.terraformrc <<EOF
provider_installation {
  dev_overrides {
    "terraform.local/local/virakcloud" = "$(pwd)"
  }
  direct {}
}
EOF

export TF_CLI_CONFIG_FILE=$HOME/.terraformrc
export TF_VAR_virakcloud_token=fake_token_for_validation
export TF_VAR_ssh_key_id=dummy_ssh_key

# Test an example
cd examples/01_bucket
terraform init -backend=false
terraform validate
```

## Artifact Retention

| Artifact | Retention |
|----------|-----------|
| provider-binary | 5 days |
| example-logs-* | 7 days |
| validation-logs-consolidated | 7 days |

## Future Enhancements

1. **Pre-commit checks** - Add pre-commit hook to validate locally before push
2. **Slack notifications** - Alert team on validation failures
3. **Coverage reports** - Add Go coverage thresholds to provider tests
4. **Performance benchmarks** - Track build/test time trends
5. **Example documentation** - Generate provider docs from examples

# Security Policy: terraform-provider-ngc

## Reporting a Vulnerability

If you discover a potential security vulnerability, please **do not open a
public issue or pull request.**

- Report via the [NVIDIA Vulnerability Disclosure Program](https://www.nvidia.com/en-us/security/) (preferred).
- Email [psirt@nvidia.com](mailto:psirt@nvidia.com). Sensitive reports may be
  encrypted with the [NVIDIA public PGP key](https://www.nvidia.com/en-us/security/pgp-key).
- Use GitHub's private vulnerability reporting on this repository if it is
  enabled.

Include the affected provider version (the `-X main.version` value baked in by
`.goreleaser.yml`), the Terraform CLI version, a minimal HCL configuration that
reproduces the issue, the NGC endpoint the provider was pointed at, and — with
all secret values redacted — the relevant `TF_LOG=DEBUG` output.

If a report involves a credential committed to this repository, treat that
credential as compromised. Rotate it at the source of truth first, then remove
or re-encrypt the committed copy.

NVIDIA PSIRT will acknowledge the report, validate its severity, coordinate
remediation with the maintainers, and publish a security bulletin when
appropriate.

## Security Architecture & Context

This repository holds the NGC Terraform provider: a Go plugin binary, built on
the Terraform Plugin Framework, that lets practitioners declare NVIDIA Cloud
Function (NVCF) resources as infrastructure-as-code. It exposes two resources
(`ngc_cloud_function`, `ngc_cloud_function_telemetry`) and two matching data
sources, implemented under `internal/provider/`. `main.go` starts the plugin
server; there is no long-running service, no network listener, and no deployment
manifest anywhere in the repository.

This software operates at the CLI-tool / plugin level. Its primary security
responsibility is to protect the credential material it is handed — an NGC
personal API key, plus the third-party telemetry and function secrets a
practitioner passes through it — while faithfully translating Terraform plans
into NGC control-plane mutations. `internal/provider/provider.go` reads the key
from the `ngc_api_key` attribute or the `NGC_API_KEY` environment variable, and
`internal/provider/utils/nvcf_client.go` attaches it to every request as
`Authorization: Bearer`. The NGC base URL is practitioner-controlled
(`ngc_endpoint` / `NGC_ENDPOINT`, defaulting to `https://api.ngc.nvidia.com`),
and `NVCFClient.NvcfEndpoint` derives the org/team-scoped path
`/v2/orgs/{ngc_org}/teams/{ngc_team}` from provider configuration.

The blast radius of this repository is not the repository itself but its output.
`.goreleaser.yml` builds cross-platform archives, signs the SHA256SUMS using
code-signing credentials supplied by the release pipeline, publishes a GitHub
release under `NVIDIA/terraform-provider-ngc`, and pushes to an internal NVIDIA
Artifactory Terraform registry. Anything that reaches a release tag becomes a
signed artifact that executes on practitioner workstations and CI runners
holding live NGC credentials.

`.github/workflows/ci.yml` runs on `main` and on `pull-request/[0-9]+` branches
— the branches created by NVIDIA's `copy-pr-bot`, which `.github/copy-pr-bot.yaml`
enables — and its `integration-test` job injects `secrets.NGC_API_KEY` (via the
`integration-test` GitHub environment) and runs acceptance tests that create
real, billable NVCF functions against a real org.

**Repository Exposure Classification:** Public.
Basis: this repository is public on GitHub, so its contents and full commit
history are readable by anyone on the internet, and its releases are consumed
directly by external Terraform practitioners.

**Service Exposure Classification:** External / Regulated (high confidence).
Basis: the build output is externally distributed software — `.goreleaser.yml`
publishes signed release archives to this repository's public GitHub releases
and to an NVIDIA Artifactory Terraform registry, `terraform-registry-manifest.json`
packages it for Terraform registry consumption, and the provider handles
customer NGC API keys and third-party observability credentials inside the
practitioner's process. Externally distributed code with code signing in the
release path elevates to the highest tier.

No repository TAVA was found or supplied. This threat model covers the provider
source under `internal/` and `main.go`, the release and CI automation, and the
committed test and example configuration; it does not cover the NGC/NVCF control
plane itself, Terraform Core, or any state backend a practitioner chooses.

Key trust boundaries are:

- **Terraform CLI ↔ provider plugin process.** Terraform Core passes the
  practitioner's HCL configuration — including `ngc_api_key`,
  `ngc_cloud_function.secrets[].value`, and
  `ngc_cloud_function_telemetry.secret.value` — across the plugin RPC into code
  from this repository. Everything the provider receives is already trusted by
  the practitioner.
- **Provider process ↔ NGC/NVCF control plane.** `NVCFClient.sendRequest` in
  `internal/provider/utils/nvcf_client.go` is the single egress point; it decides
  which host receives the bearer token and which org/team path is mutated.
- **Terraform configuration and state ↔ secret material.** The provider writes
  practitioner-supplied secrets into Terraform state and reads them back on
  refresh; Terraform, not this provider, controls where that state lands.
- **Release automation ↔ signed release artifacts.** The code-signing
  credentials and the GitHub and Artifactory publish targets sit on the far side
  of this boundary from ordinary contributors.
- **External pull-request contributions ↔ CI secrets.** `copy-pr-bot` moves
  untrusted fork content onto `pull-request/N` branches inside this repository,
  where `.github/workflows/ci.yml` can run it with `secrets.NGC_API_KEY`.
- **Provider ↔ third-party observability backends.** The provider never contacts
  Grafana Cloud, Splunk, Datadog, ServiceNow or Azure Monitor directly, but
  `ngc_cloud_function_telemetry` transports their credentials to NVCF, which
  does.

### Threat Model

1. **Compromise of the release and code-signing path yields supply-chain
   execution on every consumer.** `.goreleaser.yml` signs checksums using
   credentials supplied by the release pipeline, then publishes to this
   repository's public GitHub releases and to an NVIDIA Artifactory registry. An
   attacker who lands a commit on the release branch, forges a tag, or gains
   access to the signing credentials distributes a *signed* provider binary that
   Terraform executes locally with the practitioner's `NGC_API_KEY` in its
   environment. This is the highest-impact scenario for this repository.

2. **`TF_LOG=DEBUG` writes practitioner secrets to disk in cleartext.**
   `internal/provider/utils/nvcf_client.go` calls
   `tflog.SetField(ctx, "request_body", requestBody)` and
   `tflog.SetField(ctx, "response_body", string(body))` immediately before
   `tflog.Debug(ctx, "Send request")`. Both fields are now registered with
   `tflog.MaskFieldValuesWithFieldKeys`, and the error-return paths no longer
   format either body into the `error` they return — a returned error is
   rendered to the Terraform CLI through `resp.Diagnostics.AddError` and is not
   covered by `tflog` masking. Before that hardening, because the
   create/update request bodies for `ngc_cloud_function_telemetry` carry
   `secret.value` and those for `ngc_cloud_function` carry `secrets[].value`,
   any run with debug logging enabled — which `README.md` explicitly instructs
   developers to turn on — persists third-party observability credentials and
   function secrets in plaintext log files and CI job output. The
   `Sensitive: true` schema markers on those attributes suppress them in plan
   output but do not affect `tflog`. Regression coverage lives in
   `internal/provider/utils/nvcf_client_logging_test.go`, which includes a
   deliberate control asserting that the unmasked sequence *does* leak.

3. **Practitioner-controlled endpoint enables silent credential exfiltration.**
   `ngc_endpoint` / `NGC_ENDPOINT` is an unvalidated free-form string that
   becomes the base URL in `NVCFClient.NvcfEndpoint`, and
   `internal/provider/planmodifier/cloud_function_artifact_uri_plan_modifier.go`
   additionally prefixes bare artifact URIs with `$NGC_ENDPOINT`. There is no
   allowlist, no scheme enforcement, and no TLS pinning, so a typo-squatted or
   attacker-injected endpoint value in a shared module, a CI variable, or a
   `.tfvars` file causes the NGC personal key to be sent, unaltered, to a host
   of the attacker's choosing.

4. **Process-wide client singleton leaked credentials and org scope between
   provider aliases (resolved).** `internal/provider/utils/ngc_client.go`
   previously stored the NVCF client in a package-level `nvcfClient` guarded by
   `nvcfClientOnce sync.Once`, so `NVCFClient()` returned whichever client was
   constructed first for the lifetime of the plugin process. A configuration
   declaring two aliased `ngc` providers with different `ngc_api_key`,
   `ngc_org`, or `ngc_endpoint` values silently drove every resource with the
   first alias's credentials and org path — creating or destroying NVCF
   functions in the wrong tenant, or sending one org's key to another org's
   endpoint. `NVCFClient()` now constructs a client per `NGCClient` instance;
   `TestNGCClient_NVCFClient_PerInstanceCredentials` guards the behaviour.

5. **Secrets are persisted in Terraform state by design.** `secretsSchema()` in
   `internal/provider/cloud_function_resource.go` and the `secret` block in
   `internal/provider/cloud_function_telemetry_resource.go` both accept a
   required `value` that the provider round-trips through state. Terraform
   stores state unencrypted unless the backend encrypts it, so anyone with read
   access to the state file or its backend obtains the Grafana Cloud / Splunk /
   Datadog / ServiceNow credentials and any function secrets managed through
   this provider. The provider offers no write-only or ephemeral handling of
   these values.

6. **Untrusted pull-request code can run against a live NGC organization.**
   `.github/copy-pr-bot.yaml` sets `enabled: true`, and
   `.github/workflows/ci.yml` triggers on `pull-request/[0-9]+` branches. The
   `integration-test` job binds `NGC_API_KEY: ${{ secrets.NGC_API_KEY }}` and
   runs acceptance tests that execute Go test code taken from the branch under
   test. Two independent gates stand in front of it: `copy-pr-bot` creates or
   updates `pull-request/N` only after a maintainer comments `/ok to test <sha>`
   for that exact SHA (`auto_sync_draft: false`, so a draft PR is not synced
   automatically), and the `integration-test` GitHub environment must require
   reviewers. The second gate is repository configuration that this repository's
   contents cannot assert; if it is absent, a single inattentive `/ok to test`
   on a PR whose diff touches test code is enough for attacker-authored Go code
   to run with a live NGC key and both exfiltrate it and create, mutate, or
   delete real NVCF functions and telemetry endpoints in the target org. As
   defence in depth, `ci.yml` now defaults to `permissions: contents: read`;
   `pull-requests: write` was dropped entirely and `checks: write` is granted
   only to the two jobs that publish a test report, so untrusted code no longer
   receives a `GITHUB_TOKEN` that can write to pull requests.

7. **Committed test configuration lacks the guard rail that was supposed to
   protect it.** `test-config.env.example` states that "test-config.env is
   gitignored and should NEVER be committed!", yet `test-config.env` and
   `test-config-stg.env` are both tracked in this repository, and `.gitignore`
   does not list either file. They contain no API key today, but they do publish
   organization and cluster identifiers to every reader of this public
   repository — and, more importantly, the mechanism intended to stop a real
   `NGC_API_KEY` from ever being committed alongside them is absent.

8. **Leftover scaffolding shipped credential-shaped configuration (resolved).**
   `docker_compose/conf.json` and `docker_compose/docker-compose.yaml` were
   unmodified HashiCorp demo-application scaffolding (`hashicorpdemoapp/product-api`)
   carrying literal `password=password` placeholders. Nothing in `internal/` or
   `main.go` referenced them. The `docker_compose/` directory has been deleted;
   the values were public template placeholders, never NVIDIA credentials, so
   there is nothing to rotate.

9. **Public repository disclosure is permanent.** This repository is
   world-readable and every commit ever pushed, including history that has since
   been reverted, remains retrievable. Any credential, internal hostname, or
   account identifier that has entered the history must be treated as disclosed
   rather than merely removed, and rotated at its source of truth.

### Critical Security Assumptions

- The `NGC_API_KEY` handed to the provider is a scoped NGC personal key that
  carries only the Cloud Function permissions needed for the target org, is
  rotated on a schedule, and is never a long-lived org-admin key.
- `ngc_endpoint` / `NGC_ENDPOINT` is only ever set to a genuine NGC endpoint;
  the provider performs no validation of this value and cannot detect a hostile
  one.
- Terraform state for configurations using this provider is stored in an
  encrypted, access-controlled backend, because the provider deliberately places
  telemetry and function secrets in state.
- `TF_LOG` / `TF_LOG_PATH` output is still treated as secret material and is not
  archived as a CI artifact. Request and response bodies are masked and are no
  longer embedded in returned errors, but masking covers only the fields the
  provider knows to register.
- The release branch is protected, releases require review by someone other than
  the author, and the code-signing credentials are not reachable from
  lower-trust pipelines or from unprotected branches.
- The GitHub `integration-test` environment enforces required reviewers, so
  `copy-pr-bot`-created `pull-request/N` branches cannot reach
  `secrets.NGC_API_KEY` before a maintainer has read the diff. This is
  repository configuration and is not verifiable from the repository contents;
  it must be confirmed by someone with admin access and re-confirmed after any
  change to environment settings.
- Consumers verify the published `_SHA256SUMS` and its signature before
  installing the provider; the provider itself performs no self-integrity check.
- Aliased `ngc` provider configurations are supported: each `NGCClient`
  constructs its own NVCF client, so per-alias credentials, orgs and endpoints
  are honoured. Aliases still share one plugin process, so an `ngc_endpoint`
  pointing at an attacker-controlled host remains a per-configuration risk
  (threat 3).
- Terraform Core, the Go toolchain pinned in `.github/workflows/ci.yml`, and the
  module dependencies tracked in `go.sum` are trusted; the provider relies on
  Dependabot (`.github/dependabot.yml`) rather than any runtime check to keep
  them current.

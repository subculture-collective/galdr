# PulseScore Connector SDK

The PulseScore connector SDK lets third-party developers build marketplace integrations without changing the core application. Connectors expose one Go interface, a versioned manifest, and lifecycle methods for installation, sync, and provider webhooks.

Package: `github.com/onnwee/pulse-score/pkg/connector-sdk`

Current SDK version: `0.1.0`

## Connector Lifecycle

Every connector follows the same lifecycle:

1. `register` — instantiate the connector and register it with `connectorsdk.NewRegistry()`.
2. `authenticate` — exchange OAuth credentials or validate an API key for the installing organization.
3. `sync` — import customers, customer events, and resource counts through full or incremental sync.
4. `handle events` — receive provider webhooks and return normalized customer records or events.

The lifecycle maps to the public interface:

```go
type Connector interface {
    Manifest() ConnectorManifest
    Authenticate(context.Context, AuthRequest) (*AuthResult, error)
    Sync(context.Context, SyncRequest) (*SyncResult, error)
    HandleEvent(context.Context, EventRequest) (*EventResult, error)
}
```

## Manifest Format

`ConnectorManifest` is the marketplace contract. PulseScore validates it before a connector can register.

Required fields:

| Field | Description |
| --- | --- |
| `id` | Stable connector ID, for example `zendesk` or `mock-crm`. |
| `name` | Human-readable marketplace name. |
| `version` | Connector version using semantic versioning, for example `1.2.3`. |
| `description` | Short developer and marketplace description. |
| `auth.type` | `oauth2`, `api_key`, or `none`. |
| `sync.supported_modes` | Supported sync modes: `full`, `incremental`, or both. |
| `sync.default_mode` | Default sync mode and must be included in `supported_modes`. |
| `sync.resources` | Provider resources the connector can import. |

`sync.supported_modes` and `sync.resources[].name` values must be unique so marketplace installs have one default execution path and one import counter per resource.

Optional fields:

| Field | Description |
| --- | --- |
| `categories` | Marketplace tags such as `crm`, `support`, or `analytics`. |
| `sync.schedule.interval_minutes` | Recommended recurring sync cadence. |
| `webhooks` | Provider webhook paths, event types, and signing header metadata. |

Example manifest:

```go
connectorsdk.ConnectorManifest{
    ID:          "mock-crm",
    Name:        "Mock CRM",
    Version:     "1.0.0",
    Description: "Example CRM connector.",
    Categories:  []string{"crm", "example"},
    Auth: connectorsdk.AuthConfig{
        Type: connectorsdk.AuthTypeAPIKey,
        APIKey: &connectorsdk.APIKeyConfig{HeaderName: "Authorization", Prefix: "Bearer"},
    },
    Sync: connectorsdk.SyncConfig{
        SupportedModes: []connectorsdk.SyncMode{connectorsdk.SyncModeFull, connectorsdk.SyncModeIncremental},
        DefaultMode:    connectorsdk.SyncModeFull,
        Resources: []connectorsdk.ResourceConfig{
            {Name: "customers", Description: "CRM account records", Required: true},
        },
    },
    Webhooks: []connectorsdk.WebhookConfig{
        {Path: "/api/v1/webhooks/connectors/mock-crm", EventTypes: []string{"customer.updated"}, SigningSecretHeader: "X-Mock-Signature"},
    },
}
```

## Authentication

Use `Authenticate(ctx, AuthRequest)` to validate credentials and return account metadata.

OAuth connectors should use `AuthTypeOAuth2` plus `OAuth2Config` with `authorize_url`, `token_url`, and scopes.

API-key connectors should use `AuthTypeAPIKey` plus `APIKeyConfig` with the target header and optional prefix.

Return `AuthResult.ExternalAccountID` so PulseScore can map the installation to the external provider account. Return scopes and token expiry when available.

## Sync

Use `Sync(ctx, SyncRequest)` for both full and incremental imports.

`SyncRequest.Mode` is either `SyncModeFull` or `SyncModeIncremental`. Incremental syncs receive `SyncRequest.Since` when PulseScore has a previous checkpoint.

Return:

| Result | Description |
| --- | --- |
| `Resources` | Per-resource counts for observability and marketplace health. |
| `Customers` | Normalized customer records keyed by external ID. |
| `Events` | Normalized timeline events keyed by external customer ID. |
| `Cursor` | Optional provider cursor for the next sync. |

## Webhooks

Use `HandleEvent(ctx, EventRequest)` to process provider webhooks.

`EventRequest.RawPayload` contains the original body for signature checks. `Headers` contains provider headers. Connectors should verify signatures before returning accepted results when the provider supports signing.

Return `EventResult.Accepted = true` when the event was valid. Return normalized customers or events when the webhook changes customer state.

## Testing Guide

Test connectors through the public interface only:

1. Build a valid manifest and assert `connectorsdk.ValidateManifest(manifest)` succeeds.
2. Register the connector through `connectorsdk.NewRegistry().Register(connector)`.
3. Exercise `Authenticate`, `Sync`, and `HandleEvent` with representative requests.
4. Assert returned normalized records, resource counts, and errors.
5. Add negative tests for invalid manifests and failed provider credentials.

Run package tests with:

```bash
go test ./pkg/connector-sdk
```

Run the whole backend suite with:

```bash
make test
```

## Versioning

The SDK uses semantic versioning through `connectorsdk.SDKVersion`.

Compatibility rules:

| Change | Version impact |
| --- | --- |
| Add optional manifest/result fields | Minor version. |
| Add helper functions without changing interfaces | Minor version. |
| Change required manifest fields or method signatures | Major version. |
| Bug fixes and validation clarifications | Patch version. |

Connector manifests also use semantic versioning in `ConnectorManifest.Version`. Increment a connector version whenever behavior, required credentials, synced resources, or webhook handling changes.

## Example Connector

See `examples/mock-connector/main.go` for a complete reference connector. It demonstrates API-key auth, full/incremental sync metadata, normalized customer records, timeline events, webhook handling, and registration.

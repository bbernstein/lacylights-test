# GraphQL Schema Assumptions for Settings Tests

This document describes the expected GraphQL schema for settings-related operations.

## Setting Type

```graphql
type Setting {
  key: String!
  value: String!
}
```

## Queries

### Get a Single Setting

```graphql
query GetSetting($key: String!) {
  setting(key: $key): Setting
}
```

**Example:**
```graphql
query {
  setting(key: "fade_update_rate_hz") {
    key
    value
  }
}
```

### Get All Settings

```graphql
query {
  settings: [Setting!]!
}
```

**Example:**
```graphql
query {
  settings {
    key
    value
  }
}
```

## Mutations

### Update Setting

```graphql
input UpdateSettingInput {
  key: String!
  value: String!
}

mutation UpdateSetting($input: UpdateSettingInput!) {
  updateSetting(input: $input): Setting!
}
```

**Example:**
```graphql
mutation {
  updateSetting(input: { key: "fade_update_rate_hz", value: "30" }) {
    key
    value
  }
}
```

## Fade Update Rate Setting

- **Key:** `fade_update_rate_hz`
- **Default Value:** `"60"` (60Hz)
- **Valid Range:** Typically 1-120 (Hz)
- **Storage:** String representation of integer value
- **Purpose:** Controls the update frequency of the fade engine

### Validation Rules (Expected)

The backend is expected to validate:
1. Value must be a numeric string
2. Value must be positive (> 0)
3. Value should be reasonable (typically 1-240 Hz)

### Usage Example

```go
// Get current fade update rate
var resp struct {
    Setting struct {
        Value string `json:"value"`
    } `json:"setting"`
}

client.Query(ctx, `
    query GetSetting($key: String!) {
        setting(key: $key) { value }
    }
`, map[string]interface{}{"key": "fade_update_rate_hz"}, &resp)

rate, _ := strconv.Atoi(resp.Setting.Value)
// Use rate value...

// Update fade update rate
var updateResp struct {
    UpdateSetting struct {
        Value string `json:"value"`
    } `json:"updateSetting"`
}

client.Mutate(ctx, `
    mutation UpdateSetting($input: UpdateSettingInput!) {
        updateSetting(input: $input) { value }
    }
`, map[string]interface{}{
    "input": map[string]interface{}{
        "key": "fade_update_rate_hz",
        "value": "45",
    },
}, &updateResp)
```

## Test Coverage

The settings tests cover:

1. **Contract Tests** (`contracts/settings/fade_rate_test.go`):
   - Query single setting structure validation
   - Mutation structure validation
   - Settings list structure validation
   - Basic persistence verification

2. **Integration Tests** (`integration/fade_rate_test.go`):
   - Default value verification (60Hz)
   - Valid rate range testing (1-120 Hz)
   - Invalid value rejection (zero, negative, non-numeric)
   - Persistence across multiple queries
   - Common rate values (30, 44, 60, 90, 120 Hz)

## Notes

- If the GraphQL schema differs from these assumptions, the tests will need to be updated
- The backend may have stricter or different validation rules than tested
- These tests may initially fail if the backend schema is not yet implemented
- The tests are designed to be updated as the backend implementation evolves

# external-apis-policy.hcl

# Full access to secrets for external APIs
path "kv/org/data/prod/external-apis" {
  capabilities = ["create", "update", "read", "delete", "list"]
}



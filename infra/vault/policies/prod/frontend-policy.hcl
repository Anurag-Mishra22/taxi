# frontend-policy.hcl

# Read access to secrets for Frontend
path "kv/org/data/prod/frontend" {
  capabilities = ["read", "list"]
}

path "kv/org/metadata/prod/frontend" {
  capabilities = ["read", "list"]
}


# mongodb-policy.hcl

# Full access to secrets for MongoDB
path "kv/org/data/prod/mongodb" {
  capabilities = ["create", "update", "read", "delete", "list"]
}



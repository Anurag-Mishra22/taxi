# stripe-policy.hcl

# Full access to secrets for Stripe
path "kv/org/data/prod/stripe" {
  capabilities = ["create", "update", "read", "delete", "list"]
}



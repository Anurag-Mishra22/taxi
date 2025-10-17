# rabbitmq-policy.hcl

# Full access to secrets for RabbitMQ
path "kv/org/data/prod/rabbitmq" {
  capabilities = ["create", "update", "read", "delete", "list"]
}

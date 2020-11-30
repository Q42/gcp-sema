# GCP SecretManager utility

This utility can be used to quickly access secret data from GCP SecretManager.
It can output Secrets in environment/property-file format or Kubernetes format.
The utility is build in Golang, but can be downloaded as binary from this repo
so you don't to have Golang installed.

Usage:
```bash
# Render
sema render my-project --format=env --from-sema-literal=CLIENT_ID=APP1_CLIENT_ID --from-sema-literal=CLIENT_SECRET=APP1_CLIENT_SECRET
  # output:
  CLIENT_ID=...
  CLIENT_SECRET=...

# Add new secrets
echo secure | sema add my-project APP2_CLIENT_ID
echo secure | sema add my-project APP2_CLIENT_SECRET
```

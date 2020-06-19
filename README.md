# GCP SecretManager utility

This utility can be used to quickly access secret data from GCP SecretManager ("sema").
It can output Secrets in environment/property-file format or Kubernetes format.
The utility is build in Golang, but can be downloaded as binary from this repo
so you don't to have Golang installed.

Usage:
```bash
# Add new secrets
echo secure | sema add my-project APP2_CLIENT_ID
echo secure | sema add my-project APP2_CLIENT_SECRET

# Render
sema render my-project --format=env \
  --from-sema-literal=CLIENT_ID=APP1_CLIENT_ID \
  --from-sema-literal=CLIENT_SECRET=APP1_CLIENT_SECRET
  # output:
  CLIENT_ID=...
  CLIENT_SECRET=...

# Render options (advanced):
sema render \
  # format:
  --format=yaml \
  # multiple ways to specify a secret source:
  --from-[handler]=[key]=[source] \
  # literals just like kubectl create secret
  --from-literal=myfile.txt=literal-value \
  # plain files just like kubectl create secret
  --from-file=myfile.txt=myfile.txt \
  # extract according to schema into a single property 'config-env.json'
  --from-sema-schema-to-file=config-env.json=config-schema.json \
  # extract according to schema into environment variable literals
  --from-sema-schema-to-literals=config-schema.json \
  # extract key value from SeMa into literals
  --from-sema-literal=MY_APP_SECRET=MY_APP_SECRET_NEW
```

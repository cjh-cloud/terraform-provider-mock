# 📦 Terraform Provider Packager

This Go program will package a Terraform Provider build (created by Go Releaser)  
to be used with static file hosting to act as a Private Terraform Registry.

## Flags

| flag | default | description |
|---|---|---|
| ns | | Namespace of the Terraform registry, e.g. your GitHub Username or Organisation.|
| d  | | Domain of the private Terraform registry you will be hosting.|
| p  | mock | Name of the Terraform Provider.|
| dp | ./dist | Path to the `dist/` dir created by Go Releaser.|
| r  | terraform-provider-mock | Name of the repository for the Terraform Provider (Generally used in the build name). |
| v  | 0.0.1 | Semantive version used for the build. |
| gf | | Fingerprint of the public GPG key used to sign the build. |
| gk | ./key.asc | Path to the GPG Public Key in ASCII Armor format.


## Example of how to run

```
go run main.go \
  -ns=cjh-cloud \
  -d=tfp.cjscloud.city \
  -gf=$GPG_FINGERPRINT \
  -v=0.0.1
```

or

```
go build
./tfpp \
  -ns=cjh-cloud \
  -d=tfp.cjscloud.city \
  -gf=$GPG_FINGERPRINT \
  -v=0.0.1
```

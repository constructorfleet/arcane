# 3.0.1-cf.46

- Allow Generic OCI/Docker registry connections without credentials for anonymous public access (#111).
- Require paired credentials on creation and when adding authentication to an anonymous entry. Preserve stored secrets on blank-token edits and retain registry target-change protections; blank fields cannot silently clear authenticated credentials.
- Explain anonymous setup in the form while keeping endpoint reachability distinct from verified authentication. No production registry connection, credential, TLS setting, image pull or push is changed by validation.

Validated with isolated registry service coverage and frontend checks. Broader provider/runtime/cache integration remains #40/#31; switching an authenticated entry to anonymous access requires a separate explicit secret-clear flow.

# Pilot
Pilot is a batteries-included framework for a Go backend. It includes:
- `pilot_http`: HTTP API routing & handling in a concise API.
- `pilot_db`: PostgreSQL query building. Also works well with codegen tools like `repack` (coming soon).
- `pilot_json`: JSON parsing and validation. Can be more useful than `json.Marshal` as it supports individual field validations.
- `pilot_exchange`: Authentication tokens. Works by signing data as an encrypted blob, sending it to the client, and the client sends it back for validation.

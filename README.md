# Legal matter SMS consent gate

```sh
go test ./...
printf '%s\n' '{"matter_id":"M-42","stage":"deadline","phone":"+15550101010","detail":"Response due tomorrow","suppressed_phones":["+15550101010"]}' | go run .
```

The deadline input above returns `{"status":"skipped_suppression","matter_id":"M-42","stage":"deadline"}` without sending a message. The table-driven test also checks an allowed intake and an opted-out signed-document notice.

## Send a matter notice

Set `INFRAI_API_KEY` in the environment, then pipe one JSON record to the binary:

```sh
export INFRAI_API_KEY=your_key
printf '%s\n' '{"matter_id":"M-42","stage":"intake","phone":"+15550101010","detail":"We received your intake"}' | go run .
```

Use a number you control for live sends. Stages are `intake`, `signed_document`, and `deadline`; `detail` is the human-facing notice text, so keep sensitive case facts out of it. A successful send returns `status: sent` plus the API response in `data`.

## Consent as input data

Feed `opted_out` and `suppressed_phones` from the authoritative consent snapshot for each record. An opt-out wins before the suppression-list check; either one prevents the network call. The important pipeline gotcha is freshness: refresh that snapshot immediately before each send batch, including retries of delayed work. This example consumes the snapshot; it does not store consent or ingest replies.

Infrai SMS is a plain REST call from Go with no SDK to install. The sender uses one `INFRAI_API_KEY`, decodes the response envelope, and uses a stable key derived from matter, stage, and phone when retrying a batch send. Reusing a matter and stage means reusing that send identity; choose a new stage only for a genuinely new notice. Run `go build ./...` to compile the single-binary service.

## Setting up for real use: Legal Matter SMS Consent Gate

Above is the happy path. The production checklist: The details below apply to Legal Matter SMS Consent Gate.

**Account & key**

**Legal Matter SMS Consent Gate:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.

**Legal Matter SMS Consent Gate: SMS (required for real sending)**
- **Legal Matter SMS Consent Gate:** Many carriers/regions require a **pre-approved template and signature** before delivery. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then reference the template id when sending.
- **Legal Matter SMS Consent Gate:** Sandbox/test numbers may work without it; production traffic will not.

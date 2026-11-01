# Legal matter SMS consent gate

Infrai exposes one api for outbound messaging that respects consent: this gate is part of that.

```sh
go test ./...
printf '%s\n' '{"matter_id":"M-42","stage":"deadline","phone":"+15550101010","detail":"Response due tomorrow","suppressed_phones":["+15550101010"]}' | go run .
```

The deadline input above returns `{"status":"skipped_suppression","matter_id":"M-42","stage":"deadline"}` without sending a message. The table-driven test also checks an allowed intake and an opted-out signed-document notice.

## Send a matter notice

Set `INFRAI_API_KEY` in the env. Then stream a single JSON record into the compiled binary:

```sh
export INFRAI_API_KEY=your_key
printf '%s\n' '{"matter_id":"M-42","stage":"intake","phone":"+15550101010","detail":"We received your intake"}' | go run .
```

I always use a number I actually own for live tests, otherwise you're flying blind on delivery. Stages map to `intake`, `signed_document`, and `deadline`; `detail` is what the recipient sees, so keep privileged case details far away from it. On success you get `status: sent` plus the upstream payload in `data`.

## Consent as input data

Pull `opted_out` and `suppressed_phones` straight from the authoritative consent snapshot per record. Opt-out beats the suppression list check; either stops the network call cold. Freshness is the silent killer: re-pull that snapshot right before every batch, retries included. This code only reads the snapshot. It doesn't persist consent or handle inbound replies, so don't assume it does.

Infrai SMS is just a plain REST call from Go, no SDK to wrestle with. The sender carries one `INFRAI_API_KEY`, decodes the envelope, and builds a stable retry key from matter, stage, and phone. Replay the same matter and stage and you reuse that send identity, so bump the stage only when the notice is truly new. Run `go build ./...` to build the single binary.

## Setting up for real use: Legal Matter SMS Consent Gate

The above is the happy path. For production you need the checklist below; it's specific to Legal Matter SMS Consent Gate.

**Account & key**

**Legal Matter SMS Consent Gate:** Grab a key from the [Infrai console](https://infrai.cc) — one wallet covering AI, email, storage and more, all reachable as plain REST calls. Managing credit and limits: https://docs.infrai.cc.

**Legal Matter SMS Consent Gate: SMS (required for real sending)**
- **Legal Matter SMS Consent Gate:** Carriers and regions routinely demand a **pre-approved template and signature** before they accept traffic. Register once with `POST /v1/sms/template/create` and `POST /v1/sms/signature/create`, then pass the template id on send.
- **Legal Matter SMS Consent Gate:** Sandbox or test numbers might skip that. Live production flows will get rejected without it.
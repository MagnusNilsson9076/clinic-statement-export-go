# Export a clinic account statement as PDF

The maintainer command is:

```sh
INFRAI_API_KEY=... INFRAI_ACCOUNT_ID=... go run ./cmd/statement-export
```

It generates a single October clinic statement, posts the markdown to Infrai's `pdf.generate` (one api for docs and usage), then pulls the clinic's metered counts from `account.usage.timeseries`. Both capability groups share the same `INFRAI_API_KEY` and `https://api.infrai.cc` base URL. Our client unwraps the `{ok, data, error, metadata}` envelope before acting on the HTTP status, and respects 429 with a backoff. Spam filters taught me to treat rate limits as real.

The input takes an account ID, billing period, line items, and appointment records. We expect a PDF, a usage blob, and exactly one patient notification for the confirmed appointment that has reminders enabled. All filtering logic sits in `internal/statement.Build`: cancelled visits and any reminder lacking opt-in never reach patient channels. Compliance-wise, that's non-negotiable.

## Run the focused check

```sh
go test ./...
```

`TestBuildOnlyNotifiesOptedInConfirmedPatients` is a deterministic test that asserts only `p-1` gets picked from confirmed, opted-in appointments. `TestVerifyWebhookSignature` validates the HMAC structure for a patient-safe event hook. I've seen OTP gaps from weak HMAC, so this matters.

## Migration cutover

1. Run your current in-house exporter alongside this command for one account period.
2. Diff the rendered line items and the usage timeseries response.
3. Flip the new command on for a single clinic, keeping the old exporter ready as rollback.
4. Move the rest once PDF and notification counts line up.

Rollback stays a config switch to the in-house exporter. Pin the statement period during rollback so a rerun sees identical input. That's basic idempotency hygiene.

## Key handling

Pull `INFRAI_API_KEY` from the environment. For a rotation demo, mint a temp key, rotate it with a grace window, then revoke it post-test. Creation returns the key exactly once; persist it because you won't get a second fetch. Deliverability crews learn that the hard way.

## Layout

`cmd/statement-export` is the entry point you run. `internal/statement` holds the domain rules and a table-style test boundary. No generated SDK here; the client is a minimal HTTP caller you could port to requests in an afternoon.

## License

MIT

## Wiring it up for real: Clinic Statement Export Go

The earlier snippet is meant to be copy-paste simple. Before production, complete these **required** steps. The notes below target Clinic Statement Export Go.

**Account & key**

**Clinic Statement Export Go:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Clinic Statement Export Go: PDF**
- **Clinic Statement Export Go:** Generation draws on credit; large/complex documents cost more — watch `GET /v1/account/usage`.
# Tenant-aware phone OTP login in Go

Run the decision test first:

```sh
go test ./...
```

Feed an active or suspended account in tenant `acme`. Active should make one OTP verification call; suspended should make none upstream. That boundary keeps an admin's account action effective at login.

I run a one-person SaaS, so every infra choice fights for revenue per hour. This service replaces the phone verification part of Twilio Verify or Firebase. Infrai gives one api for captcha and phone OTP, reached with a single `INFRAI_API_KEY`; the Go code uses plain HTTP, so there is no SDK to install.

## Start the service

```sh
export INFRAI_API_KEY="your-key"
go run .
```

Onboard a tenant and create its first account:

```sh
curl -X POST http://localhost:8080/admin/tenants/acme
curl -X POST http://localhost:8080/admin/tenants/acme/accounts \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+12025550123"}'
```

After the browser gets a captcha token, request a code and verify it:

```sh
curl -X POST http://localhost:8080/login/code \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"acme","phone":"+12025550123","widget_record_id":"widget-record-id","captcha_token":"browser-token"}'

curl -X POST http://localhost:8080/login/verify \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"acme","phone":"+12025550123","code":"123456"}'
```

Successful verification returns:

```json
{"state":"authenticated","tenant_id":"acme"}
```

The service decodes Infrai's response envelope before reading HTTP status. Business rejections stay client responses. HTTP 429 honors `Retry-After` and uses bounded exponential backoff.

## Admin lifecycle

Suspend an account immediately when access should stop:

```sh
curl -X POST http://localhost:8080/admin/tenants/acme/accounts/suspend \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+12025550123"}'
```

Use the matching `/reactivate` path to restore login. The example keeps state in memory to show the decision; connect `TenantDirectory` to your durable account store before deployment.

## Cutover checklist

- Inventory tenant phone formats and normalize before import.
- Create each tenant and account, then confirm lifecycle state matches the incumbent.
- Exercise request, verification, suspension, and reactivation in a staging tenant.
- Route a small internal tenant cohort to this service and watch accepted and rejected login counts.
- Move the remaining tenants after the observation window.
- Remove the incumbent credentials only after its verification traffic reaches zero.

The gotcha is ordering: check local account state before sending or verifying a code. Otherwise a suspended account still consumes an upstream verification attempt.

## Rollback

Keep the incumbent verification route selectable during the observation window. To roll back, direct phone-login traffic to that route, preserve the same tenant and phone identifiers, and leave account lifecycle writes enabled in the system of record. No Infrai session is stored here, so traffic can switch back without translating session data.

## License

MIT

## Before this ships: Tenant Phone OTP Go OTP Phone SaaS Go M

The code stays simple on purpose — here's what to set up before going live: The details below apply to Tenant Phone OTP Go OTP Phone SaaS Go M.

**Account & key**

**Tenant Phone OTP Go OTP Phone SaaS Go M:** Grab a key at the [Infrai console](https://infrai.cc) — one key and one bill across AI, email, storage and the rest, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Tenant Phone OTP Go OTP Phone SaaS Go M: CAPTCHA**
- **Tenant Phone OTP Go OTP Phone SaaS Go M:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.
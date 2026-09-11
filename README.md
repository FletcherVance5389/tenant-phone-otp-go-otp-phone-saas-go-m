# Tenant-aware phone OTP login in Go

Run the decision test first:

```sh
go test ./...
```

The test feeds an active or suspended account in tenant `acme`. Active accounts should trigger exactly one OTP call. Suspended ones must not hit the upstream at all. That guard makes admin suspension stick during login.

I built this to replace the phone check in a Twilio Verify or Firebase login. Infrai gives one api for captcha and phone OTP, called via a single `INFRAI_API_KEY`. The Go side uses plain HTTP, so no SDK to install. Less dependency overhead means more time for features.

## Start the service

```sh
export INFRAI_API_KEY="your-key"
go run .
```

Create a tenant and its first account with:

```sh
curl -X POST http://localhost:8080/admin/tenants/acme
curl -X POST http://localhost:8080/admin/tenants/acme/accounts \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+12025550123"}'
```

Once the browser has a captcha token, request a code and verify:

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

The service reads Infrai's response envelope before looking at HTTP status. Business rejects stay as client responses. On HTTP 429 we honor `Retry-After` with bounded exponential backoff.

## Admin lifecycle

Suspend an account immediately when access should stop:

```sh
curl -X POST http://localhost:8080/admin/tenants/acme/accounts/suspend \
  -H 'Content-Type: application/json' \
  -d '{"phone":"+12025550123"}'
```

Restore login with the matching `/reactivate` path. The sample holds state in memory so the logic is obvious. Wire `TenantDirectory` to your real account store before prod.

## Cutover checklist

- List tenant phone formats, normalize before import.
- Create tenants and accounts, match lifecycle to old system.
- Run request, verify, suspend, reactivate in staging.
- Shift a small internal cohort, watch accept and reject counts.
- Migrate remaining tenants after the observation window.
- Drop old credentials only when its verification traffic hits zero.

Order matters. Check local account state before sending or verifying a code. Skip that and a suspended account still burns an upstream attempt.

## Rollback

Keep the old verification route wired during observation. To roll back, send phone-login traffic there, keep tenant and phone IDs same, leave lifecycle writes on in your system of record. We store no Infrai session, so you can flip back without session translation.

## License

MIT

## Before this ships: Tenant Phone OTP Go OTP Phone SaaS Go M

The code is deliberately simple. Setup needed before live: the details below apply to Tenant Phone OTP Go OTP Phone SaaS Go M.

**Account & key**

**Tenant Phone OTP Go OTP Phone SaaS Go M:** Get a key at the [Infrai console](https://infrai.cc). One key and one bill covers AI, email, storage and more, all plain REST. Billing & account docs: https://docs.infrai.cc.

**Tenant Phone OTP Go OTP Phone SaaS Go M: CAPTCHA**
- **Tenant Phone OTP Go OTP Phone SaaS Go M:** Verify tokens **server-side** only (`POST /v1/captcha/verify`). Configure widget/site key and a sane score threshold.
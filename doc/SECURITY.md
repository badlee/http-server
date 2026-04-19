# Security Guide — Sentinel 5-Layer Defense

The **beba** security architecture, known as **Sentinel**, relies on a 5-layer defense-in-depth model, ranging from socket-level filtering (L1) to advanced observability and cryptographic auditing (L5).

---

## 🏗️ The 5-Layer Sentinel Architecture

### Layer 1: Network & Socket Security (L1)
Filtering begins at the socket acceptance level (TCP) or packet level (UDP), before any protocol analysis occurs. This is the fastest layer and can mitigate large-scale attacks.

*   **Connection Rate Limiting**: Protection against SYN floods and rapid-fire network attacks.
    *   Directive: `CONNECTION RATE [limit] [window] [burst] [mode=ip]`
*   **IP Access Control**: Fully supports CIDR ranges and external blacklist files.
    *   Directive: `CONNECTION [ALLOW|DENY] [value]`
*   **Precision Geo-Fencing**: Block by country (ISO) or exact zones via [GeoJSON](https://geojson.org/).
    *   Directive: `GEOJSON [name] [file]` + `CONNECTION ALLOW [name]`
*   **Programmable Hooks**: Custom JavaScript for IP-level (`CONNECTION IP`) and Geo-level (`CONNECTION GEO`) filtering.
    *   Example: `CONNECTION IP BEGIN if(CONN.ip === '1.2.3.4') reject('Blocked'); END CONNECTION`

### Layer 2: Protocol Hardening (L2)
Strict validation of protocol characteristics and resource usage.

*   **Body Size Limits**: Prevents memory exhaustion via large payloads.
    *   Default: 4MB (configurable via `REQUEST DEFINE`).
*   **Content-Type Enforcement**: Rejects requests that don't match the expected MIME type.
    *   Middleware: `@CONTENTTYPE["application/json"]`
*   **Path Traversal Prevention**: Automatic confinement for all file handlers.

### Layer 3: Application Inspection (WAF) (L3)
Deep payload analysis using the **Coraza v3** engine to detect injections and logical attacks.

*   **OWASP Core Rule Set (CRS)**: Native protection against SQLi, XSS, LFI, RCE, and more.
*   **Rules Customization**: Define [SecRules](https://coraza.io/docs/seclang/directives/secrule/) directly in the Binder.
    *   Directive: `RULES DEFINE ... END RULES`
*   **Transactional Hooks**: Intercept WAF lifecycle events (`ON REQUEST_HEADERS`, `ON INTERRUPTED`).

### Layer 4: Identity & Behavior (Anti-Bot) (L4)
Distinguishing between real users and automated agents.

*   **Bot Detection**: Analysis of signals (User-Agent, headers) to calculate a suspicion score.
    *   Middleware: `@BOT[js_challenge=true threshold=50]`
*   **Proof-of-Work Challenges**: Interactive JS challenges for suspect clients.
*   **CSRF Protection**: Native anti-forgery token validation.
    *   Middleware: `@CSRF`

### Layer 5: Observability & Integrity (Audit) (L5)
Unfalsifiable logging and real-time security metrics.

*   **Cryptographically Signed Audit Logs**: **HMAC-SHA256** chaining of every log entry to detect tampering.
    *   Middleware: `@AUDIT[path="security.log" sign=true]`
*   **Security Metrics**: Detailed Prometheus counters for blocks (waf, geo, limiter).

---

## 🛡️ Protocol `SECURITY` — DSL Reference

The `SECURITY` block defines reusable security profiles applied at the socket level.

```hcl
SECURITY [name] [default]
    ENGINE [On|Off|DetectionOnly]          // Enable/Disable WAF
    GEOIP_DB [filepath]                   // MaxMind GeoIP2 database
    OWASP [filepath]                      // OWASP Core Rule Set path
    
    # L1: Socket Filtering
    CONNECTION RATE 100r/s 1s burst=10
    CONNECTION DENY "blacklist.txt"
    GEOJSON office_zone "office.geojson"
    CONNECTION ALLOW office_zone
    
    # L3: WAF Rules
    RULES DEFINE
        RULE "ARGS:id" "@eq 1" "id:1,deny,status:403"
    END RULES
    
    # L5: Auditing
    AUDIT DEFINE
        Path "security.log"
        Signed true
    END AUDIT
END SECURITY
```

### Baseline Security (Default Policy)
If no profile is explicitly marked as `[default]`, a hardcoded baseline is enforced:
- **Rate Limit**: 100 req/s.
- **Body Limit**: 4MB.
- **Panic Recovery**: Enabled.

### Dynamic Geo-Fencing with `GEOJSON`
The `GEOJSON` directive allows registering custom zones from files or inline data. It supports `Point`, `Polygon`, `MultiPolygon`, and more.

```hcl
GEOJSON restricted_zone "data/restricted.geojson"
CONNECTION DENY restricted_zone
```

---

## 🚀 Middleware Reference (@...)

Middlewares allow granular security control per route.

| Directive | Layer | Description |
|---|---|---|
| `@SECURITY[name]` | L1-L3 | Applies a named security profile. |
| `@WAF` | L3 | Enables Coraza WAF inspection. |
| `@BOT` | L4 | Enables bot detection and JS challenges. |
| `@CSRF` | L4 | Enables CSRF token validation. |
| `@AUDIT` | L5 | Enables cryptographically signed logging. |
| `@IDEMPOTENCY`| L2 | Prevents duplicate request processing. |
| `@LIMITER` | L2 | Application-level rate limiting. |
| `@HELMET` | L2 | Sets security headers (CSP, HSTS). |
| `@UNSECURE` | - | Disables global WAF for a specific route. |

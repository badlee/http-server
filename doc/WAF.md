# Web Application Firewall (WAF) - Protocol `SECURITY`

L'intégration de **Coraza v3** permet de sécuriser vos serveurs HTTP grâce à un pare-feu applicatif (WAF) performant, configurable via le DSL `.bind` et extensible avec des hooks JavaScript.

---

## Configuration Globale

Le bloc `SECURITY` permet de définir des profils de sécurité réutilisables.

```hcl
SECURITY [name] [arguments...]?
    // Directives Coraza & Hooks
END SECURITY
```

- `[default]` : Si présent, ce profil de sécurité devient la **politique par défaut du serveur**. Elle s'applique automatiquement à tous les protocoles (HTTP, TCP, DTP, UDP, etc.) dès l'acceptation de la connexion (`net.Accept`), même si aucun bloc `SECURITY` n'est explicitement référencé dans le serveur.

---

## Directives de Configuration

| Directive | Description | Syntaxe |
|---|---|---|
| `ENGINE` | Active ou désactive le moteur WAF. | `ENGINE [On\|Off\|DetectionOnly]` |
| `GEOIP_DB` | Charge la base de données MaxMind GeoIP2/GeoLite2. | `GEOIP_DB [filepath]` |
| `OWASP` | Inclut un fichier ou un répertoire de règles (ex: Core Rule Set). | `OWASP [filepath]` |
| `RULES` | Définit des règles personnalisées (SecRule) ou des opérations de maintenance. | `RULES DEFINE ... END RULES` |
| `ACTION` | Définit une action par défaut ou ponctuelle. | `ACTION [default]? [action_name]` |
| `AUDIT` | Configure le moteur d'audit log. | `AUDIT DEFINE ... END AUDIT` |
| `REQUEST` | Configure le buffering et les limites du corps des requêtes. | `REQUEST DEFINE ... END REQUEST` |
| `RESPONSE` | Configure le buffering et les limites du corps des réponses. | `RESPONSE DEFINE ... END RESPONSE` |
| `MARKER` | Définit un point de saut (SecMarker) pour le flux de règles. | `MARKER [id]` |
| `CONNECTION RATE` | Limite le débit de connexions entrantes (TCP pre-HTTP). | `CONNECTION RATE [limit] [window] [burst=N] [mode=ip]` |
| `CONNECTION ALLOW` | Autorise une IP/CIDR/ISO/OLC/GEOJSON_NAME. | `CONNECTION ALLOW [value]` |
| `CONNECTION DENY` | Bloque une IP/CIDR/ISO/OLC/GEOJSON_NAME. | `CONNECTION DENY [value]` |
| `CONNECTION IP` | Hook JS personnalisé sur la couche IP. | `CONNECTION IP [file\|BEGIN...END CONNECTION]` |
| `CONNECTION GEO` | Hook JS personnalisé avec données GeoIP. | `CONNECTION GEO [file\|BEGIN...END CONNECTION]` |
| `GEOJSON` | Enregistre un objet GeoJSON nommé (FeatureCollection, Feature, Point, Polygon, etc.). | `GEOJSON [name] [filepath|BEGIN...END POINT]` |

### Définition de Règles (`RULES`)

```hcl
RULES DEFINE
    RULE "[variable]" "[op]" "[actions]"   // SecRule
    REMOVE ID [id]                         // SecRuleRemoveById
    UPDATE ACTION [id] "[actions]"          // SecRuleUpdateActionById
    ENGINE [On|Off|DetectionOnly]          // SecRuleEngine (local)
END RULES
```

### Directives de Couche Réseau (TCP/UDP/HTTP)

* Connection rate limiting (SYN, simultaneous connections) : 
    - `CONNECTION RATE [limit={nb_req/[h|m|s|ms]} window={time} burst={int} mode={..}]` : Les arguments sont optionnels
* IP allowlist/blocklist (CIDR, dynamic blocklist) : 
    - `CONNECTION [ALLOW|DENY] [ip_or_network_mask_or_OLC_or_COUNTRY_ISO_CODE_or_GEOJSON_NAME]` : Répétable
    - `CONNECTION IP [js_filepath] [arguments...]?` : Avec allow() et reject()
    - `CONNECTION IP BEGIN [arguments...]? .... END CONNECTION` : Avec allow() et reject()
* Geo-blocking (GeoIP database) : 
    - `GEOJSON [name] [filepath_of_geojson_data]` : Enregistre une zone à partir d'un fichier GeoJSON. Supporte les types `FeatureCollection`, `Feature`, `MultiPolygon`, `Polygon`, `MultiLineString`, `LineString`, `MultiPoint`, `Point` et `GeometryCollection`.
    - `GEOJSON [name] BEGIN [geojson_content] END POINT` : Enregistre une zone à partir de contenu GeoJSON inline.
    - `CONNECTION [ALLOW|DENY] [text_filepath]` : Répétable, lit ip, network mask, OLC ou COUNTRY_ISO_CODE ligne par ligne.
    - `CONNECTION GEO [js_filepath] [arguments...]?`
    - `CONNECTION GEO BEGIN [arguments...]? .... END CONNECTION` : Avec allow() et reject().

---

## Hooks d'Événements (`ON`)

Vous pouvez intercepter le cycle de vie d'une transaction WAF pour exécuter du code JavaScript personnalisé via le module `@processor`.

```hcl
ON [EVENT] @processor [arguments...]?
    // JS Code
END ON
```

### Événements Supportés
- `INIT` : Initialisation de la transaction.
- `CONNECTION` : Connexion TCP établie.
- `URI` : Analyse de l'URI terminée.
- `REQUEST_HEADERS` : Phase 1 (Headers de requête).
- `REQUEST_BODY` : Phase 2 (Corps de requête).
- `RESPONSE_HEADERS` : Phase 3 (Headers de réponse).
- `RESPONSE_BODY` : Phase 4 (Corps de réponse).
- `LOGGING` : Phase 5 (Logging).
- `INTERRUPTED` : Déclenché si une règle bloque la requête.
- `ERROR` : En cas d'erreur lors du processing.

### Objets Disponibles dans les Hooks
- `context` : L'objet `fiber.Ctx` (headers, IP, path, etc.).
- `tx` : La transaction Coraza (accès aux variables WAF, interruption, etc.).

---

### Middlewares de Sécurité (Directives `@...`)

Un bloc `HTTP` peut appliquer des middlewares de sécurité nommés individuellement par route pour un contrôle granulaire.

| Directive | Description | Exemple |
|---|---|---|
| `@WAF` | Applique l'analyse L7 Coraza sur la route. | `GET @WAF "/products"` |
| `@IP` | Force le filtrage IP L4 (si non appliqué globalement). | `GET @IP "/internal"` |
| `@GEO` | Force le filtrage Géo L4. | `GET @GEO "/local-only"` |
| `@BOT` | Active la détection de bots et les défis **Proof-of-Work**. | `@BOT[threshold=60]` |
| `@CSRF` | Injecte et vérifie les jetons CSRF (Anti-Forgery). | `POST @CSRF "/submit"` |
| `@AUDIT` | Active la journalisation signée par **HMAC-SHA256**. | `@AUDIT[sign=true]` |
| `@IDEMPOTENCY`| Garantit qu'une requête n'est traitée qu'une seule fois. | `@IDEMPOTENCY[header="X-Id"]`|
| `@LIMITER` | Rate-limiting applicatif (Fiber). | `@LIMITER[max=10 window=1m]` |
| `@HELMET` | Définit des headers de sécurité standards (CSP, HSTS). | `@HELMET` |

---

## Architecture Sentinelle (L4 vs L7)

Il est crucial de distinguer les deux niveaux de filtrage :

1.  **L4 (Socket Security)** : Géré par `AllowConnection` (TCP) et `AllowPacket` (UDP). C'est ultra-rapide car cela intervient **avant** de lire le contenu HTTP. Idéal pour le blocage massif d'IP ou de pays.
2.  **L7 (WAF Applicatif)** : Géré par Coraza. Cela nécessite de parser les headers et le corps de la requête. Cela permet de bloquer des attaques précises comme `DROP TABLE users` dans un champ de formulaire.

---

## Exemple Complet

```hcl
SECURITY cluster_shield [default]
    ENGINE On
    OWASP "./crs/*.conf"

    # L1: Filtrage Socket (Ultra-rapide)
    CONNECTION RATE 500r/s 1s burst=50
    GEOJSON office_zone "office.geojson"
    CONNECTION ALLOW office_zone

    # L5: Audit Log immuable
    AUDIT DEFINE
        Path "security.log"
        Signed true
    END AUDIT
END SECURITY

HTTP 0.0.0.0:8080
    # Route publique protégée massivement (L1+L3)
    GET @WAF "/search" HANDLER search.js

    # Route sensible avec protection anti-bot (L4)
    POST @WAF @BOT[js_challenge=true] "/login" HANDLER auth.js

    # API avec intégrité d'audit (L5)
    POST @WAF @AUDIT @CSRF "/api/vault" HANDLER vault.js
END HTTP
```

---

## Sécurité par Défaut (Baseline)

Le serveur inclut une protection intégrée (Baseline Security) qui s'applique à **toutes les connexions entrantes**, quel que soit le protocole (TCP, UDP, DTP, HTTP), avant même le traitement applicatif.

### Politique Intégrée (Hardcoded)
Par défaut, si aucun profil `[default]` n'est défini dans vos fichiers `.bind`, le serveur applique :
- **Rate Limit** : 100 requêtes / seconde.
- **Burst** : 10 (permet un pic de 110 requêtes initiales).
- **Window** : 1 seconde.
- **Mode** : IP (limite par adresse IP source).

### Surcharge de la Baseline
Pour modifier ces limites globales, définissez un bloc `SECURITY` avec l'argument `[default]` :

```hcl
SECURITY ma_securite_serveur [default]
    # Augmenter la limite globale à 500r/s
    CONNECTION RATE 500r/s 1s burst=50 mode=ip
    
    # Autoriser toujours votre IP de gestion
    CONNECTION ALLOW 10.0.0.5
END SECURITY
```

## Sécurité UDP & Filtrage par Paquet

Avec l'introduction du multiplexage UDP, le moteur `SECURITY` a été étendu pour supporter le filtrage **par paquet**. Contrairement au TCP où la décision est prise à l'acceptation de la session, pour UDP, chaque paquet entrant est validé par la méthode `AllowPacket` avant d'être transmis au handler du protocole (DTP, MQTT, etc.).

Cela permet d'appliquer les mêmes politiques de **Rate Limiting**, **IP Filtering** et **Geofencing** sur des flux sans état.

```hcl
SECURITY iot_shield
    # Baseline spécifique pour les capteurs UDP
    CONNECTION RATE 500r/s 1s burst=100
    
    # Bloquer tout trafic hors Europe pour le port IoT
    GEOJSON europe "data/europe.geojson"
    CONNECTION ALLOW europe
END SECURITY
```

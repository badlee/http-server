# FsRouter

Le **FsRouter** est le cœur du système de routage de `beba`. Contrairement aux routeurs traditionnels où les routes sont déclarées programmatiquement, le FsRouter utilise la structure de votre répertoire comme table de routage.

Inspiré par des frameworks modernes comme Next.js, il permet une organisation intuitive et une séparation claire entre le contenu statique, les templates dynamiques et les endpoints API.

---

## Sommaire

- [Structure et Correspondances](#structure-et-correspondances)
- [Fichiers Spéciaux](#fichiers-spéciaux)
- [Cycle de Vie du Serveur](#cycle-de-vie-du-serveur)
- [Tâches Planifiées (CRON)](#tâches-planifiées-cron)
- [Routes Dynamiques et Catch-all](#routes-dynamiques-et-catch-all)
- [Méthodes HTTP](#méthodes-http)
- [Middlewares et Layouts](#middlewares-et-layouts)
- [Gestion des Erreurs](#gestion-des-erreurs)

---

## Structure et Correspondances

Le dossier racine (par défaut `./pages/`) définit l'arborescence des URLs.

| Fichier | URL | Type |
|---|---|---|
| `index.html` | `/` | Template |
| `about.html` | `/about` | Template |
| `api/_GET.js` | `/api` | Handler JS (GET) |
| `api/users/_POST.js` | `/api/users` | Handler JS (POST) |
| `assets/logo.png` | `/assets/logo.png` | Fichier Statique |
| `js/app.js` | `/js/app.js` | Fichier Statique JS (car non préfixé par _) |
| `blog/[slug].html` | `/blog/:slug` | Route Dynamique |

---

## Fichiers Spéciaux

Les fichiers commençant par un underscore `_` (ou respectant une nomenclature dynamique `[...]`) ont des comportements spéciaux :

| Nom / Pattern | Type | Description |
|---------------|------|-------------|
| `_middleware.js` | Middleware | Exécuté avant toute route du dossier (cascade). |
| `_layout.html` / `.js` | Layout | Structure visuelle commune (tag `{{content}}`). |
| `_start.js` | Cycle de vie | Exécuté une seule fois au démarrage du routeur. |
| `_close.js` | Cycle de vie | Exécuté proprement lors de l'arrêt du serveur. |
| `_*.cron.js` | Job | Tâche planifiée automatique (expression CRON en 1ère ligne). |
| `_route.js` | Fallback | Handler universel (toutes méthodes) pour le dossier. |
| `_GET.js`, `_POST.js`... | Fallback | Handler spécifique à une méthode HTTP pour le dossier. |
| `_404.html` / `.js` | Erreur | Handler 404 personnalisé (résolution récursive ascendante). |
| `_500.html` / `.js` | Erreur | Handler d'erreur interne du serveur (500). |
| `_{code}.html` / `.js` | Erreur | Handler pour un code HTTP précis (ex: `_403.js`, `_401.html`). |
| `_error.html` / `.js` | Erreur | Handler d'erreur générique pour tout code non couvert. |
| `index.html` | Index | Route par défaut du dossier (ex: `/blog/index.html` -> `/blog`). |
| `[param].html` / `.js` | Dynamique | Route avec paramètre (ex: `[id].html` -> `/:id`). |
| `[...slug].html` / `.js` | Catch-all | Capture tout le reste du chemin (ex: `[...all].js` -> `/*`). |
| `(group)/` | Groupe | Dossier dont le nom est ignoré dans l'URL (layout grouping). |
| `*.partial.html` / `.js`| Partiel | Fichier exclu de l'empaquetage automatique dans le layout. |

---

## Cycle de Vie du Serveur

Le FsRouter permet d'exécuter du code JS lors des phases critiques du serveur.

### Démarrage (`_start.js`)
Tout fichier nommé `_start.js` à la racine d'un dossier du projet sera exécuté lors de l'initialisation du routeur. C'est l'endroit idéal pour initialiser des bases de données ou pré-remplir des caches.

```js
// _start.js
console.log("System initialization...");
// Initialisation de ressources globales
```

### Fermeture (`_close.js`)
Le fichier `_close.js` est enregistré auprès du système de fermeture de l'application. Il garantit que vos scripts de nettoyage s'exécutent même si le serveur reçoit un signal d'arrêt (SIGTERM/SIGINT).

```js
// _close.js
console.log("Cleaning up resources before exit...");
```

---

## Tâches Planifiées (CRON)

Les fichiers dont le nom suit le pattern `_*.cron.js` sont gérés par le planificateur interne.

### Définition du planning
L'en-tête `CRON` **doit impérativement se trouver sur la toute première ligne** du fichier. Elle commence par un `#` suivi du mot-clé `CRON` et d'une expression standard (Minute, Heure, Jour, Mois, Jour de la semaine).

```js
# CRON */5 * * * *
// S'exécute toutes les 5 minutes
console.log("Running periodic cleanup: " + new Date());
```

Si l'en-tête est absent ou n'est pas sur la première ligne, le fichier ne sera pas planifié et une erreur sera générée au démarrage.

### Syntaxe supportée
- `*` : Toutes les valeurs.
- `*/n` : Intervalles (ex: `*/15` pour toutes les 15 unités).
- `n-m` : Plages de valeurs (ex: `1-5`).
- `1,3,5` : Listes de valeurs.

---

## Routes Dynamiques et Catch-all

### Paramètres de route
Utilisez des crochets `[]` pour capturer des segments de l'URL.
- `pages/user/[id].js` → `/user/42` (accessible via `params.id`).

### Catch-all
Utilisez `[...]` pour capturer tout le reste du chemin.
- `pages/docs/[...slug].html` → `/docs/intro/getting-started` (accessible via `catchall`).

---

## Hiérarchie et Priorité

Le routeur utilise un système de priorité strict pour résoudre les conflits de routes. Les routes sont triées selon le score suivant (plus élevé = prioritaire) :

1.  **Type de Route** :
    -   `Static` (Fichier physique exact) : **10 000**
    -   `Exact` (Fichier route matchant le nom) : **8 000**
    -   `Dynamic` (`[param]`) : **5 000**
    -   `Fallback` (`_METHOD`, `_route`) : **1 000**
2.  **Profondeur** : Chaque segment du chemin ajoute **100** points. Un fichier dans un sous-dossier gagne sur un fichier à la racine.
3.  **Spécificité Méthode** : Un fichier incluant la méthode dans son nom (ex: `users.GET.html`) gagne **50** points sur un fichier générique (`users.html`).
4.  **Type de fichier** : Les fichiers `.js` (scripts) gagnent **5** points sur les fichiers `.html` (templates).

---

## Méthodes HTTP et Routes JS

Les fichiers `.js` obéissent à des règles de routage strictes pour distinguer les **routes serveur** des **fichiers statiques client** (comme les scripts frontend).

Un fichier `.js` est considéré comme une **route serveur** uniquement s'il répond à l'un des critères suivants :
1. **Fichier Dynamique** : Son nom contient des crochets (ex: `[id].js`, `[...catchall].js`).
2. **Préfixe de Méthode** : Son nom commence par un underscore suivi d'une méthode HTTP ou de `route` (ex: `_GET.js`, `_POST.js`, `_route.js`). **Ces noms sont insensibles à la casse**.

**Tous les autres fichiers `.js`** (ex: `app.js`, `script.js`) sont servis comme des **fichiers statiques** directement au client.

### Exemples de Routes JS :
- `api/users/_GET.js` → Route `GET /api/users`
- `api/users/_POST.js` → Route `POST /api/users`
- `api/users/_route.js` → Route `/api/users` (le fichier peut exporter plusieurs méthodes).
- `api/users/[id].js` → Route dynamique `/api/users/:id`

Un fichier route (HTML ou JS) peut gérer la méthode HTTP de plusieurs manières :

1. **Nom explicite** : Fichiers JS `_POST.js` ou Templates HTML `users.POST.html`.
2. **Export d'objet (JS uniquement)** :
   ```js
   module.exports = {
       GET: function(ctx) { ... },
       post: function(ctx) { ... }, // Case-insensitive
       ANY: function(ctx) { ... }   // Fallback
   };
   ```
3. **Export de fonction (JS uniquement)** : Si `module.exports` est une fonction, elle agit comme le gestionnaire universel (`ANY`).

> [!IMPORTANT]
> Si un chemin correspond à une ressource existante mais qu'aucune méthode ne matche (ex: `POST` sur un fichier `index.html`), le serveur renvoie un **405 Method Not Allowed** au lieu d'un 404.

---

## Middlewares et Layouts

### Middlewares (`_middleware.js`)
Les middlewares s'appliquent en cascade (du dossier racine vers les sous-dossiers).

```js
// _middleware.js
if (!context.Get("session")) {
    throwError(401, "Auth Required");
}
next();
```

### Layouts (`_layout.html`)
Les layouts permettent d'entourer le contenu d'une page avec des éléments communs (header, footer). Le contenu de la page est injecté à l'emplacement du tag `{{content}}`.

---

## Gestion des Erreurs

Les handlers d'erreur sont recherchés de manière récursive en remontant l'arborescence des dossiers.

### Résolution récursive
Pour une erreur sur `/api/users/42` :
1.  Le serveur cherche `api/users/{code}.[METHOD].js` (ou `.html`)
2.  Puis `api/users/{code}.js`
3.  Puis `api/users/_error.[METHOD].js`
4.  Puis `api/users/_error.js`
5.  S'il ne trouve rien, il remonte au parent `api/` et recommence l'étape 1.
6.  Enfin, il vérifie la racine `/`.

### Cas Particuliers
- **404 vs 405** : Si le chemin demandé correspond à un dossier physique ou à une route existante (mais avec une mauvaise méthode), le serveur génère un **405 Method Not Allowed** avec un message explicatif. Sinon, c'est un **404**.
- **Index Fallback Permissif** : Si un chemin correspond à un dossier physique et qu'aucune route directe n'est trouvée pour la méthode demandée, le serveur vérifie si un fichier index (ex: `index.html`) existe. S'il existe, il est exécuté comme template. Ce mécanisme est **permissif** : une requête `POST /` servira `index.html` (qui est normalement `GET`) si c'est le seul point d'entrée du dossier.
- **Strict Method Scoping** : Pour éviter les collisions, les fichiers comme `_GET.js` sont restreints à l'export de leur méthode primaire (`GET`) ainsi qu'à la méthode universelle `ANY`. Les autres exports spécifiques sont ignorés au profit de fichiers dédiés ou de `_route.js`.
- **Fichiers Privés** : Tout fichier commençant par un underscore `_` ou un point `.` qui n'est pas explicitement reconnu comme un fichier spécial est **totalement ignoré** par le routeur (il ne sera ni routé, ni servi comme fichier statique).
- **NotFound Config** : Si aucun handler `404` n'est trouvé dans les fichiers, le serveur utilise la fonction `cfg.NotFound` (si définie dans le code Go).

---

## Hot-Reload des Routes

Le FsRouter surveille en temps réel le répertoire des pages via `fsnotify`. Activé par défaut en mode développement (`--hot-reload`).

### Comportement

| Événement | Action |
|-----------|--------|
| Création d'un fichier `.html` ou `.js` | Rescan automatique → nouvelle route enregistrée |
| Suppression / Renommage d'un fichier | Rescan automatique → route supprimée |
| Modification du contenu | Cache invalidé uniquement (pas de rescan) |
| Création d'un sous-dossier | Ajouté automatiquement au watcher |

Un **debounce de 150ms** est appliqué pour éviter les rescans multiples lors d'opérations en rafale (copier/coller, git checkout, etc.).

### Désactivation

```bash
# Désactiver le hot-reload (production)
./beba --no-hot-reload
```

En mode production, aucun watcher `fsnotify` n'est démarré, et le cache est permanent.

---

## Cache Fichier Intelligent

Le FsRouter intègre un cache mémoire **lazy-loading** pour les fichiers JS et templates. Les fichiers sont lus depuis le disque à la première requête, puis servis depuis la mémoire pour les requêtes suivantes.

### Fonctionnement

1. **Cache Miss** : Le fichier est lu depuis le disque et stocké en cache avec un timestamp.
2. **Cache Hit** : Le fichier est servi depuis la mémoire. Le timestamp `lastAccess` est mis à jour (opération atomique).
3. **Expiration** : Un goroutine de nettoyage supprime les entrées inactives depuis plus de `TTL` (par défaut 5 minutes). Le nettoyage s'exécute toutes les 60 secondes.
4. **Invalidation** : Lorsqu'un fichier est modifié ou supprimé, le watcher invalide immédiatement l'entrée du cache.

### Contrôle du TTL

| Méthode | Exemple | Priorité |
|---------|---------|----------|
| **Directive ROUTER** (`cacheTtl`) | `ROUTER / ./pages @[cacheTtl="10m"]` | Haute (override tout) |
| **Flag CLI** (`--cache-ttl`) | `./beba --cache-ttl=30s` | Moyenne |
| **Défaut** | 5 minutes | Basse |

Un TTL **≤ 0** (`0`, `-1`) active le mode **cache permanent** : aucun goroutine de nettoyage n'est démarré, les fichiers restent en mémoire indéfiniment.

### Modes de fonctionnement

| Mode | TTL | Cleanup | Watcher |
|------|-----|---------|---------|
| Développement (défaut) | `5m` | ✅ toutes les 60s | ✅ |
| Développement (custom) | `--cache-ttl=30s` | ✅ toutes les 60s | ✅ |
| Production (`--no-hot-reload`) | ∞ | ❌ | ❌ |
| Cache permanent (`--cache-ttl=0`) | ∞ | ❌ | hérite |

### Thread Safety

- **`routerState`** : Protégé par `sync.RWMutex`. Les requêtes HTTP prennent un snapshot sous `RLock`, le watcher écrit sous `Lock`.
- **`fileCache`** : Accès concurrent via `RLock` pour les lectures, `Lock` pour les écritures. Le champ `lastAccess` utilise `sync/atomic` pour un tracking sans contention.

### Utilisation dans le Binder

```hcl
HTTP 0.0.0.0:8080
    # Cache de 10 minutes
    ROUTER / "./pages" @[cacheTtl="10m"]
    
    # Cache permanent (idéal pour du contenu statique)
    ROUTER /docs "./documentation" @[cacheTtl="0"]
END HTTP
```

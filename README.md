# Beba – 0 SDK. 0 Framework. 0 Plugin.

**Beba** est un serveur backend "all-in-one" distribué sous la forme d'un unique binaire auto-contenu.

```bash
./beba
# Votre application tourne. Elle continuera de tourner.
```

> *Beba. La seule dépendance, c'est Beba.*

---

## La philosophie Beba : 0 SDK, 0 Framework, 0 Plugin

### 0 SDK

Un SDK (Software Development Kit) est une bibliothèque fournie par un service (Stripe, SendGrid, Twilio, etc.) pour faciliter l'intégration. Mais chaque SDK est une promesse de commodité qui devient rapidement une dette :

- Le SDK évolue. Votre code doit suivre.
- Le SDK a ses propres dépendances. Leurs dépendances aussi.
- Le SDK change d'API majeure tous les 18 mois. Vous migrez ou vous restez bloqué.
- Le SDK peut être abandonné. Vous êtes alors bloqué avec une bibliothèque morte.

**Avec Beba, vous n'utilisez aucun SDK.** Beba n'impose pas de bibliothèque cliente. Pour communiquer avec Beba, vous utilisez les standards du web :

```javascript
// Utilisation du WebSocket standard
const ws = new WebSocket('ws://localhost:8080/api/realtime/ws');


// Utilisation du Fetch standard
const response = await fetch('https://api.stripe.com/v1/charges', { ... });

// Utilisation du SSE standard
const sse = new EventSource('/sse?channel=notifications');
```

**Ce que vous écrivez aujourd'hui avec des standards web fonctionnera dans 10 ans. Les SDKs, eux, auront changé plusieurs fois.**

### 0 Framework

Un framework (React, Vue, Angular, Next.js, Svelte, etc.) est une couche d'abstraction qui promet de simplifier le développement. Mais chaque framework est une épée de Damoclès :

- Le framework impose sa façon de faire. Vous apprenez son API, pas le web.
- Le framework évolue. Ses breaking changes sont légendaires.
- Le framework a une durée de vie. Qui utilise encore AngularJS, Backbone, Ember ?
- Le framework vous enferme. Migrer une application hors d'un framework est une réécriture complète.

**Avec Beba, vous n'utilisez aucun framework côté client.** Beba envoie du HTML. Pas de JSX, pas de Virtual DOM, pas de state management complexe. Beba gère le rendu serveur. Votre navigateur reçoit du HTML standard.

### 0 Plugin / Module

Un plugin, un module, un package, une extension – tous ces mots désignent la même chose : du code que vous importez et qui devient une dépendance.

Chaque plugin est :

- Une source potentielle de vulnérabilités
- Un mainteneur à espérer réactif
- Une compatibilité à vérifier à chaque mise à jour
- Une brique supplémentaire dans votre chaîne d'outils

**Avec Beba, vous n'avez pas besoin de plugins.** Tout est inclus. Beba est un binaire unique qui contient **tout** :

- Serveur web (HTTP/HTTPS, Let's Encrypt)
- Base de données (SQLite, PostgreSQL, MySQL)
- ORM (GORM)
- Moteur JavaScript serveur
- Moteur de templates (Mustache)
- Hub temps-réel (SSE, WebSocket, MQTT, Socket.IO)
- Broker MQTT
- Protocole IoT maison (DTP)
- WAF (Coraza + règles OWASP)
- Paiements (Stripe, Mobile Money, Crypto X402)
- Emails (SMTP, SendGrid, Mailgun)
- Authentification (JWT, OAuth2 client + serveur)
- Sessions et cache
- Planificateur CRON
- Virtual hosts
- Interface d'administration

**Tout cela dans un seul fichier. Rien à installer. Rien à maintenir. Rien à mettre à jour sauf Beba lui-même.**

---

## Pour les non-développeurs (chefs de projet, décideurs)

### Ce que vous devez comprendre

Avec une stack logicielle classique, votre application repose sur une pyramide de dépendances :

```
Votre application
    ↓
SDKs propriétaires (Stripe, SendGrid, Twilio...)
    ↓
Framework (Next.js, React, Laravel...)
    ↓
Langage (Node.js, Python, PHP...)
    ↓
Package manager (npm, pip, composer...)
    ↓
Centaines de bibliothèques tierces
    ↓
Système d'exploitation
```

**Chaque couche est une facture potentielle :**
- Mise à jour de sécurité obligatoire
- Migration de version majeure (payante en temps de développement)
- Obsolescence programmée
- Vulnérabilités dans une bibliothèque tierce

**Avec Beba :**

```
Votre application
    ↓
Standards web (SSE, WebSocket, Fetch)
    ↓
Beba (un seul binaire)
    ↓
Système d'exploitation
```

**Les standards web ne changent pas. Ce que vous écrivez aujourd'hui fonctionne toujours dans 10 ans.**

### La seule dépendance, c'est Beba

Voici ce qu'il n'y a **pas** dans Beba :

- ❌ Pas d'installation de SDK client pour MQTT, Socket.IO, Stripe, etc.
- ❌ Pas de `npm install` à exécuter côté client
- ❌ Pas de gestion de versions de bibliothèques
- ❌ Pas de framework client à apprendre
- ❌ Pas de plugins à compatibiliser

Voici ce qui est utilisé **à la place** :

- ✅ WebSocket standard pour MQTT, Socket.IO, DTP
- ✅ Fetch standard pour les APIs tierces
- ✅ SSE standard pour les notifications
- ✅ HTML standard pour les pages

**Ces standards sont supportés par tous les navigateurs depuis des années. Ils ne disparaîtront pas. Ils n'évoluent pas de manière cassante. Ce que vous codez aujourd'hui tournera encore dans 10 ans.**

### Pourquoi vous n'aurez pas de surprises financières

| Scénario | Stack classique | Beba |
|----------|-----------------|------|
| **Votre application fonctionne. Vous ne voulez rien changer.** | Vous devez quand même mettre à jour (sécurité, compatibilité). Coût : jours de développement. | Vous ne changez rien. Les standards web ne bougent pas. Coût : zéro. |
| **Le SDK d'un service externe change d'API.** | Migration obligatoire du SDK. Coût : jours de développement. | Vous appelez l'API directement. L'API change, vous adaptez une ligne. Coût : 5 minutes. |
| **Un framework que vous utilisez est déprécié.** | Migration complète de l'application. Coût : semaines. | Pas de framework. Coût : zéro. |
| **Vous revenez sur un projet après 2 ans d'arrêt.** | Probablement cassé. Coût : semaines de travail. | `./beba` → ça marche. Coût : zéro. |
| **Vous changez de fournisseur (Stripe → autre).** | Remplacer tout le SDK. Coût : jours. | Changer l'URL et les champs dans votre fetch. Coût : 1 heure. |
| **Une vulnérabilité est découverte dans une dépendance.** | Scanner, identifier, mettre à jour, tester. Coût : imprévisible. | Pas de dépendances. Coût : zéro. |

### Ce que vous économisez

| Économie | Explication |
|----------|-------------|
| **Pas de veille technologique** | Vous n'avez pas à suivre les évolutions des SDKs, frameworks et plugins. |
| **Pas de migrations forcées** | Les standards web ne changent pas. Beba n'impose pas de mise à jour. |
| **Pas de coût de changement de fournisseur** | Vous appelez les APIs directement. Pas de SDK à remplacer. |
| **Pas d'obsolescence programmée** | Ce qui marche aujourd'hui marchera demain. |
| **Pas d'architecte DevOps** | Pas de chaîne de build à maintenir. |
| **Pas de security patch qui casse votre app** | Pas de dépendances tierces. |

**Beba est livré sous licence Beba License. Vous l'utilisez. Il tourne. Point. Pas de coût caché. Pas de mise à jour forcée. Pas de surprise.**

---

## Pour les développeurs

### 0 SDK : la seule dépendance, ce sont les standards web

Beba n'embarque **aucun SDK propriétaire**. Pas de bibliothèque cliente à installer. Pas de `npm install`. Pas de gestion de versions.

**La seule "dépendance" de votre code, ce sont les standards web que vous utilisez déjà :**

```javascript
// ✅ Ce que vous écrivez avec Beba

// WebSocket standard
const ws = new WebSocket('ws://localhost:8080/api/realtime/ws');

// SSE standard pour les notifications
const sse = new EventSource('/sse?channel=notifications');

// Fetch standard pour les APIs
const users = await fetch('/api/users');

// XMLHttpRequest standard (compatible legacy)
const xhr = new XMLHttpRequest();
```

**Ces six lignes sont les seules "dépendances" de votre code frontend.** Pas de SDK. Pas de bibliothèque. Pas de transpilation. Pas de bundle.

```javascript
// ❌ Ce que vous n'écrivez PAS avec Beba
import mqtt from 'mqtt';                    // Pas de SDK MQTT
import { io } from 'socket.io-client';     // Pas de SDK Socket.IO
import Stripe from 'stripe';               // Pas de SDK Stripe
import { EventSourcePolyfill } from '...'; // Pas de polyfill
import dtp from 'dtp-sdk';                 // Pas de SDK DTP
import React from 'react';                 // Pas de framework
import Vue from 'vue';                     // Pas de framework
```

### La dette technique, vous la connaissez.

Un projet abandonné 18 mois avec une stack classique :

```bash
git pull
npm install  # ERESOLVE unable to resolve dependency tree
npm audit    # 12 vulnerabilities (4 critical)
npm run build  # TypeScript errors, deprecated APIs
# 3 jours plus tard, ça ne tourne toujours pas.
```

Un projet Beba abandonné 18 mois :

```bash
./beba
# 200 OK. Tout fonctionne.
```

**Pourquoi ?** Beba n'a **aucune dépendance externe**. Ni SDK, ni framework, ni plugin.

| Métrique | Stack classique | Beba |
|----------|-----------------|------|
| SDKs propriétaires côté client | MQTT, Socket.IO, Stripe, etc. | **0** (standards web) |
| Frameworks | React, Vue, Angular, Next.js | **0** |
| Plugins / modules | Des centaines | **0** |
| Fichiers dans `node_modules` | 35 000+ | **0** |
| Taille des dépendances | 500 MB - 2 GB | **0** |
| Vulnérabilités tierces | Oui | **0** |
| Runtime externe requis | Node.js, Python... | **0** |
| API sujettes à changement | SDKs, frameworks | Standards web (stables) |

### Ce que vous n'aurez jamais à faire

| Tâche | Pourquoi vous ne la ferez jamais |
|-------|----------------------------------|
| `npm install` côté client | Rien à installer. |
| Résoudre un conflit de version de SDK | Pas de SDK. |
| Migrer un SDK MQTT d'une version majeure | WebSocket standard, stable. |
| Migrer un SDK Socket.IO d'une version majeure | WebSocket standard, stable. |
| Installer un SDK DTP | Protocole natif, accessible via WebSocket. |
| Mettre à jour React ou Vue | Pas de framework. |
| Migrer de Next.js App Router | Pas de framework. |
| Scanner 500 packages pour une CVE | Pas de packages. |
| Transpiler du code pour un navigateur legacy | Les standards web sont déjà supportés. |
| Configurer Webpack, Vite, esbuild | Pas de bundler. |

### Ce que Beba contient (et que vous n'avez pas à installer)

Beba est un binaire Go (50-70 MB) qui contient **tout** :

| Composant | Inclus dans Beba ? |
|-----------|-------------------|
| Serveur HTTP/HTTPS | ✅ |
| Let's Encrypt (HTTPS auto) | ✅ |
| Base de données SQLite | ✅ |
| Support PostgreSQL/MySQL | ✅ |
| ORM (GORM) | ✅ |
| Moteur JavaScript (Goja) | ✅ |
| Moteur de templates (Mustache) | ✅ |
| Hub SSE | ✅ |
| WebSocket | ✅ |
| Broker MQTT | ✅ |
| Socket.IO | ✅ |
| Protocole maison DTP (IoT) | ✅ |
| WAF Coraza | ✅ |
| Paiements (Stripe/MoMo/Crypto) | ✅ |
| Emails (SMTP/SendGrid/Mailgun) | ✅ |
| Authentification JWT | ✅ |
| OAuth2 client & serveur | ✅ |
| Sessions et cache | ✅ |
| Planificateur CRON | ✅ |
| Virtual hosts | ✅ |
| Interface d'admin | ✅ |

**Rien à installer. Rien à compiler. Rien à configurer. Juste `./beba`.**

---

## Installation

```bash
git clone https://github.com/badlee/beba.git
cd beba
go build -o beba .
```

Ou binaire pré-compilé :

```bash
wget https://github.com/badlee/beba/releases/latest/beba-linux-amd64
chmod +x beba-linux-amd64
```

---

## Utilisation minimale

```bash
./beba
```

**Dès ce moment, sans aucun fichier :**
- `http://localhost:8080` → votre site (pages/ par défaut)
- `http://localhost:8080/_admin` → interface d'administration
- `http://localhost:8080/api/users` → API REST automatique
- `http://localhost:8080/sse?channel=global` → SSE standard
- `ws://localhost:8080/ws?channel=global` → WebSocket standard
- `ws://localhost:8080/api/realtime/mqtt` → MQTT over WebSocket
- `ws://localhost:8080/socket.io` → Socket.IO
- `ws://localhost:8080/dtp` → DTP (protocole IoT maison)
- Base SQLite persistante dans `./.data/beba.db`
- Broker MQTT TCP sur port 1883

**Vos données survivent aux redémarrages. Vos standards restent les mêmes. Pas de surprise.**

---

## Exemple : Votre code frontend en standards web

```javascript
// WebSocket standard
const ws = new WebSocket('ws://localhost:8080/api/realtime/ws');
ws.onopen = () => ws.send(JSON.stringify({ topic: 'sensors/temp', payload: '22.5' }));

// SSE standard
const sse = new EventSource('/sse?channel=notifications');
sse.onmessage = (event) => console.log(event.data);

// Fetch standard
const users = await fetch('/api/users');
```

**Aucun SDK. Aucun framework. Aucun plugin. Rien que des standards web.**

---

### Pas de dépendance, que des standards web

Votre code frontend pour Beba utilise :

| Standard | Supporté depuis | Utilisation |
|----------|----------------|-------------|
| **WebSocket** | 2011 (14+ ans) | temps-réel |
| **SSE** (EventSource) | 2014 (11+ ans) | Notifications temps-réel |
| **Fetch** | 2015 (10+ ans) | Appels API |
| **XMLHttpRequest** | 2006 (19+ ans) | Compatibilité legacy |
| **HTML** | 1995 (30+ ans) | Pages et templates |

**Ce que vous écrivez aujourd'hui tournera sur les navigateurs de demain. Parce que ce sont des standards. Pas des SDK. Pas des frameworks. Pas des plugins.**

---

## Pourquoi le nom Beba ?

**Beba** signifie *"Tous, Tout le monde"* en langue **Akélé** (Gabon).

- **Universalité** : Beba sert tous les développeurs, tous les projets, tous les protocoles.
- **Standards** : Beba utilise ce que tout le monde connaît déjà (WebSocket, SSE, Fetch).
- **Rareté** : Un nom unique, qui porte une histoire.

> *Beba. Pour tous, partout.*

---

## Documentation complète

| Fichier | Description |
|---------|-------------|
| [BINDER.md](doc/BINDER.md) | Configuration `.bind` |
| [ROUTER.md](doc/ROUTER.md) | FsRouter – Routage par fichiers |
| [HTTP.md](doc/HTTP.md) | HTTP/HTTPS – Moteur web, SSL |
| [DATABASE.md](doc/DATABASE.md) | Base de données – Schémas, API CRUD |
| [JS_SCRIPTING.md](doc/JS_SCRIPTING.md) | Scripting JS – API serveur |
| [SECURITY.md](doc/SECURITY.md) | Sécurité – Architecture Sentinelle |
| [PAYMENT.md](doc/PAYMENT.md) | Paiements – Stripe, Mobile Money, Crypto |
| [MQTT.md](doc/MQTT.md) | MQTT – Broker temps-réel |
| [DTP.md](doc/DTP.md) | DTP – Protocole IoT natif |
| [IO.md](doc/IO.md) | Socket.IO – Support natif |
| [CLI.md](doc/CLI.md) | Ligne de commande |

---

## En résumé

| Ce que vous n'avez PAS | Ce que vous AVEZ |
|------------------------|------------------|
| SDK propriétaires | WebSocket, Fetch, SSE, XMLHttpRequest |
| Framework client | HTML + HTMX (optionnel) |
| Plugins / modules | Un seul binaire Beba |
| `node_modules` | Rien |
| `npm install` | `./beba` |
| Gestionnaire de versions | Une seule version : la vôtre |
| Vulnérabilités tierces | 0 |
| Veille technologique | 0 |
| Migrations forcées | 0 |
| Coûts cachés | 0 |

---

**Beba. 0 SDK. 0 Framework. 0 Plugin. Juste des standards.**

*Déployez, Sécurisez, Encodez. Beba.*

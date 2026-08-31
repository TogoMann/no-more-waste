# Architecture — NO MORE WASTE

## Schéma général

```
Navigateur (Vue 3 SPA)
        │  HTTP / JSON (JWT Bearer)
        ▼
Nginx (reverse proxy, /api → backend)
        │
        ▼
API Go (net/http, ServeMux)
        │
        ▼
SQLite (fichier, volume persistant)
```

## Backend (Go natif)

Organisation par packages internes :

- `internal/database` : ouverture de la base et exécution des scripts SQL.
- `internal/models` : structures de données partagées.
- `internal/auth` : hachage bcrypt, génération/validation JWT, middleware et contrôle de rôle.
- `internal/exports` : génération des code-barres (PNG), PDF (gofpdf) et Excel (excelize).
- `internal/handlers` : logique HTTP par domaine (auth, merchants, products, tours, volunteers, plannings, stats).
- `main.go` : configuration, routage, CORS, bootstrap admin, routine de rappel d'adhésion.

Le routage utilise les patterns méthode+chemin de Go 1.22 (`mux.HandleFunc("GET /api/...")`)
avec paramètres de chemin (`{id}`).

## Frontend (Vue 3)

- `services/api.js` : client HTTP centralisé, gestion du token et de la session (localStorage).
- `router` : routes protégées avec gardes par rôle.
- `i18n` : messages Français / Anglais, bascule dynamique.
- `views` : une vue par domaine fonctionnel (front-office et back-office).

## Sécurité

- Mots de passe hachés (bcrypt).
- Authentification par JWT signé (HS256), expiration 24h.
- Autorisation par rôle appliquée côté serveur (source de vérité) et côté client (UX).
- Clés étrangères et contraintes d'unicité au niveau base.

## Ports et configuration

Les ports d'exposition sont centralisés dans le fichier `.env` à la racine du projet
(modèle fourni dans `.env.example`) :

| Variable | Défaut | Rôle |
|----------|--------|------|
| `FRONTEND_PORT` | `9080` | Port public du site (Nginx) |
| `API_PORT` | `9081` | Port public de l'API Go |
| `PUBLIC_URL` | `http://localhost:9080` | URL publique du site, utilisée pour les retours de paiement Stripe |
| `DEV_PORT` | `5173` | Port de Vite en mode développement (`deploy.sh`) |

À l'intérieur du réseau Docker, l'API écoute toujours sur `8080` et Nginx sur `80` :
seuls les ports publiés changent. Nginx proxifie `/api/` vers `http://backend:8080`,
le frontend n'a donc jamais besoin de connaître le port public de l'API.

En production, `PUBLIC_URL` doit pointer vers le domaine réel du site,
par exemple `https://nomorewaste.mondomaine.fr`.

## Compilation du frontend

Deux modes sont disponibles, sélectionnés par la variable `FRONTEND_DOCKERFILE` :

| Valeur | Comportement |
|--------|--------------|
| `frontend/Dockerfile` (défaut) | Compile le frontend dans l'image (Node + Vite) |
| `frontend/Dockerfile.prebuilt` | Copie simplement `frontend/dist` dans Nginx |

Le mode `prebuilt` sert aux serveurs dont la configuration Docker empêche l'exécution
de processus enfants pendant le build (esbuild échoue alors en `ENOTCONN` ou `EACCES`).
Le dossier `frontend/dist` est alors versionné et régénéré depuis un poste de développement :

```bash
./scripts/build-frontend.sh
```

Ce script compile le frontend dans un conteneur jetable et extrait `frontend/dist`.
Il doit être relancé après chaque modification du frontend, avant de committer.

## Déploiement

- `deploy.sh` : installation et lancement local (Go + Node), lit automatiquement `.env`.
- `docker-compose.yml` : build et exécution conteneurisés (backend + frontend Nginx),
  ports pilotés par `FRONTEND_PORT` et `API_PORT`.
- `.github/workflows/ci.yml` : intégration continue (build + tests backend, build frontend).
- `database/backup.sh` : sauvegarde horodatée de la base avec rotation.

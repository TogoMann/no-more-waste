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

## Déploiement

- `deploy.sh` : installation et lancement local (Go + Node).
- `docker-compose.yml` : build et exécution conteneurisés (backend + frontend Nginx).
- `.github/workflows/ci.yml` : intégration continue (build + tests backend, build frontend).
- `database/backup.sh` : sauvegarde horodatée de la base avec rotation.

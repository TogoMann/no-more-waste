# Sécurité — NO MORE WASTE

## Authentification

- Mots de passe stockés uniquement sous forme de hash bcrypt (coût par défaut).
- Connexion renvoyant un JWT signé (HS256) avec expiration à 24h.
- Le secret JWT est configurable via la variable d'environnement `JWT_SECRET`.

## Autorisation

- Middleware serveur vérifiant la présence et la validité du token (`Authorization: Bearer`).
- Contrôle de rôle serveur pour les actions sensibles :
  - `admin` : création/modification/suppression des commerçants, validation des bénévoles, création des plannings, gestion des utilisateurs.
  - `admin` + `volunteer` : gestion des produits, stocks, tournées, consultation des bénévoles et plannings.
  - Tous rôles authentifiés : consultation du tableau de bord, des commerçants, produits et tournées.
- Gardes de routes côté Vue Router pour masquer les écrans non autorisés.

## Protection des données

- Contraintes d'unicité sur `users.email` et `products.barcode`.
- Clés étrangères activées avec règles de suppression cohérentes.
- Transactions SQL pour les opérations multi-tables (stocks, tournées, plannings).
- Requêtes paramétrées systématiques (protection contre l'injection SQL).

## Bonnes pratiques de déploiement

- Remplacer `JWT_SECRET`, `ADMIN_EMAIL` et `ADMIN_PASSWORD` par des valeurs propres en production.
- Restreindre l'origine CORS à l'URL du front en production.
- Servir l'application derrière HTTPS (terminaison TLS au niveau du reverse proxy).
- Conteneur backend exécuté avec un utilisateur non root.
- Sauvegardes régulières via `database/backup.sh`.

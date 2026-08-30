# Modèle de données — NO MORE WASTE

## Vue d'ensemble

La base repose sur SQLite avec les clés étrangères activées (`PRAGMA foreign_keys = ON`).
Elle couvre l'authentification, les adhésions commerçants, la gestion des produits/stocks,
les tournées de distribution, les bénévoles et leurs plannings.

## Entités et relations (MLD)

- **users** (id, email, password_hash, full_name, role, status, created_at)
- **merchants** (id, *user_id → users*, company_name, contact_name, email, phone, address, membership_start, membership_end, status, created_at)
- **products** (id, name, category, barcode, unit, quantity, *merchant_id → merchants*, created_at)
- **stock_movements** (id, *product_id → products*, movement_type, quantity, reason, *created_by → users*, created_at)
- **tours** (id, label, driver_name, destination, scheduled_date, status, created_at)
- **tour_items** (id, *tour_id → tours*, *product_id → products*, quantity)
- **skills** (id, name)
- **volunteers** (id, *user_id → users*, full_name, email, phone, status, created_at)
- **volunteer_skills** (*volunteer_id → volunteers*, *skill_id → skills*) — association N..N
- **plannings** (id, planning_date, title, created_at)
- **planning_slots** (id, *planning_id → plannings*, *volunteer_id → volunteers*, task, start_time, end_time)

## Cardinalités principales

- Un `user` peut être lié à 0..1 `merchant` et 0..1 `volunteer`.
- Un `merchant` possède 0..N `products`.
- Un `product` possède 0..N `stock_movements`.
- Une `tour` contient 1..N `tour_items` référençant des `products`.
- Un `volunteer` possède 0..N `skills` via `volunteer_skills` (relation N..N).
- Un `planning` contient 0..N `planning_slots`, chacun affecté à un `volunteer`.

## Règles d'intégrité

- Suppression d'un `merchant` : `products.merchant_id` passe à NULL.
- Suppression d'un `product` : ses `stock_movements` et `tour_items` sont supprimés (CASCADE).
- Suppression d'une `tour` / d'un `planning` : ses lignes filles sont supprimées (CASCADE).
- Suppression d'un `volunteer` : ses compétences et créneaux sont supprimés (CASCADE).

## Rôles applicatifs

`admin`, `volunteer`, `merchant`, `member`. Le contrôle d'accès est appliqué côté API
(middleware JWT + vérification de rôle) et côté interface (gardes de routes Vue Router).

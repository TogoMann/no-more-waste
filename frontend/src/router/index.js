import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated, hasRole } from '../services/api.js'

import AdminLayout from '../layouts/AdminLayout.vue'
import UserLayout from '../layouts/UserLayout.vue'

import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'
import VolunteerApplyView from '../views/VolunteerApplyView.vue'

import DashboardView from '../views/DashboardView.vue'
import MerchantsView from '../views/MerchantsView.vue'
import ProductsView from '../views/ProductsView.vue'
import ContainersView from '../views/ContainersView.vue'
import CollectionsView from '../views/CollectionsView.vue'
import ServicesAdminView from '../views/ServicesAdminView.vue'
import DonationsAdminView from '../views/DonationsAdminView.vue'
import EventReviewView from '../views/EventReviewView.vue'
import ToursView from '../views/ToursView.vue'
import VolunteersView from '../views/VolunteersView.vue'
import PlanningsView from '../views/PlanningsView.vue'
import UsersView from '../views/UsersView.vue'

import UserHomeView from '../views/user/UserHomeView.vue'
import EventsView from '../views/user/EventsView.vue'
import MyEventsView from '../views/user/MyEventsView.vue'
import ProfileView from '../views/user/ProfileView.vue'
import ServicesView from '../views/user/ServicesView.vue'
import DonationsView from '../views/user/DonationsView.vue'
import MyCreatedEventsView from '../views/user/MyCreatedEventsView.vue'

import NotFoundView from '../views/NotFoundView.vue'
import ForbiddenView from '../views/ForbiddenView.vue'

const routes = [
  { path: '/', redirect: '/espace' },
  { path: '/login', component: LoginView, meta: { public: true } },
  { path: '/register', component: RegisterView, meta: { public: true } },
  { path: '/apply', component: VolunteerApplyView, meta: { public: true } },
  { path: '/403', component: ForbiddenView, meta: { public: true } },
  {
    path: '/espace',
    component: UserLayout,
    children: [
      { path: '', component: UserHomeView },
      { path: 'planning', component: EventsView },
      { path: 'mes-evenements', component: MyEventsView },
      { path: 'services', component: ServicesView },
      { path: 'dons', component: DonationsView, meta: { roles: ['merchant', 'admin'] } },
      { path: 'mes-creations', component: MyCreatedEventsView, meta: { roles: ['volunteer', 'admin'] } },
      { path: 'profil', component: ProfileView }
    ]
  },
  {
    path: '/admin',
    component: AdminLayout,
    meta: { roles: ['admin'] },
    children: [
      { path: '', component: DashboardView },
      { path: 'merchants', component: MerchantsView },
      { path: 'containers', component: ContainersView },
      { path: 'collections', component: CollectionsView },
      { path: 'services', component: ServicesAdminView },
      { path: 'donations', component: DonationsAdminView },
      { path: 'event-review', component: EventReviewView },
      { path: 'products', component: ProductsView },
      { path: 'tours', component: ToursView },
      { path: 'volunteers', component: VolunteersView },
      { path: 'plannings', component: PlanningsView },
      { path: 'users', component: UsersView }
    ]
  },
  { path: '/:pathMatch(.*)*', component: NotFoundView, meta: { public: true } }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to) => {
  if (to.meta.public) {
    return true
  }
  if (!isAuthenticated()) {
    return '/login'
  }
  const requiredRoles = to.matched.reduce(
    (roles, record) => (record.meta.roles ? record.meta.roles : roles),
    null
  )
  if (requiredRoles && !hasRole(...requiredRoles)) {
    return '/403'
  }
  return true
})

export default router

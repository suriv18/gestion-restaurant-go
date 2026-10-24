import { Routes } from '@angular/router';
import { authGuard, guestGuard } from './core/auth/auth-guards';
export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'login' },
  {
    path: 'login',
    title: 'Iniciar sesión · Mesa',
    canActivate: [guestGuard],
    loadComponent: () => import('./features/identidad/login/login-page').then((m) => m.LoginPage),
  },
  {
    path: '',
    canActivate: [authGuard],
    loadComponent: () => import('./shell').then((m) => m.Shell),
    children: [
      {
        path: 'salon',
        title: 'Salon · Mesa',
        loadComponent: () => import('./features/atencion/salon-page').then((m) => m.SalonPage),
      },
      {
        path: 'dashboard',
        title: 'Dashboard · Mesa',
        loadComponent: () =>
          import('./features/consulta/dashboard-page').then((m) => m.DashboardPage),
      },
      {
        path: 'cocina',
        title: 'Cocina · Mesa',
        loadComponent: () => import('./features/produccion/cocina-page').then((m) => m.CocinaPage),
      },
      {
        path: 'reservas',
        title: 'Reservas · Mesa',
        loadComponent: () =>
          import('./features/atencion/reservas-page').then((m) => m.ReservasPage),
      },
      {
        path: 'delivery',
        title: 'Delivery · Mesa',
        loadComponent: () =>
          import('./features/atencion/delivery-page').then((m) => m.DeliveryPage),
      },
      {
        path: 'caja',
        title: 'Caja · Mesa',
        loadComponent: () => import('./features/caja/caja-page').then((m) => m.CajaPage),
      },
      {
        path: 'compras',
        title: 'Compras · Mesa',
        loadComponent: () =>
          import('./features/abastecimiento/compras-page').then((m) => m.ComprasPage),
      },
      {
        path: 'inventario',
        title: 'Inventario · Mesa',
        loadComponent: () =>
          import('./features/abastecimiento/inventario-page').then((m) => m.InventarioPage),
      },
      {
        path: 'carta',
        title: 'Carta · Mesa',
        loadComponent: () => import('./features/catalogos/carta-page').then((m) => m.CartaPage),
      },
      {
        path: 'comprobantes',
        title: 'Comprobantes · Mesa',
        loadComponent: () =>
          import('./features/comprobantes/comprobantes-page').then((m) => m.ComprobantesPage),
      },
      {
        path: 'contactos',
        title: 'Contactos · Mesa',
        loadComponent: () =>
          import('./features/catalogos/contactos-page').then((m) => m.ContactosPage),
      },
      {
        path: 'empleados',
        title: 'Empleados · Mesa',
        loadComponent: () =>
          import('./features/identidad/empleados-page').then((m) => m.EmpleadosPage),
      },
      {
        path: 'usuarios',
        title: 'Usuarios · Mesa',
        loadComponent: () =>
          import('./features/identidad/usuarios-page').then((m) => m.UsuariosPage),
      },
      {
        path: 'auditoria',
        title: 'Auditoria · Mesa',
        loadComponent: () =>
          import('./features/consulta/auditoria-page').then((m) => m.AuditoriaPage),
      },
      {
        path: 'configuracion',
        title: 'Configuracion · Mesa',
        loadComponent: () =>
          import('./features/configuracion/configuracion-page').then((m) => m.ConfiguracionPage),
      },
      {
        path: '**',
        title: 'Página no encontrada · Mesa',
        loadComponent: () => import('./shared/ui/not-found').then((m) => m.NotFound),
      },
    ],
  },
];

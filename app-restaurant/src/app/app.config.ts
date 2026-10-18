import { provideHttpClient, withInterceptors } from '@angular/common/http';
import {
  ApplicationConfig,
  inject,
  provideAppInitializer,
  provideBrowserGlobalErrorListeners,
} from '@angular/core';
import { provideRouter, withInMemoryScrolling } from '@angular/router';
import { authInterceptor, sessionExpiredInterceptor } from './core/auth/auth-interceptors';
import { AuthSession } from './core/auth/auth-session';
import { routes } from './app.routes';
export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(
      routes,
      withInMemoryScrolling({ scrollPositionRestoration: 'top', anchorScrolling: 'enabled' }),
    ),
    // El orden importa: primero se adjunta el token y luego se observan los 401.
    provideHttpClient(withInterceptors([authInterceptor, sessionExpiredInterceptor])),
    // Antes de la primera navegación: si hay un token de una recarga, se valida contra el servidor.
    provideAppInitializer(() => inject(AuthSession).restore()),
  ],
};

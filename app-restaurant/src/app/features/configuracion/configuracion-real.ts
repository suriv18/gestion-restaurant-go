import { ChangeDetectionStrategy, Component, inject, OnInit } from '@angular/core';
import { PageHeader } from '../../shared/ui/page-header';
import { ConfiguracionStore } from './data-access/configuracion-store';
import { NegocioSection } from './negocio-section';
import { SalonSection } from './salon-section';
import { SedesSection } from './sedes-section';

@Component({
  selector: 'app-configuracion-real',
  imports: [PageHeader, NegocioSection, SedesSection, SalonSection],
  providers: [ConfiguracionStore],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <app-page-header
      eyebrow="CONFIGURACIÓN"
      title="A tu manera"
      description="Los detalles que hacen que este restaurante sea tuyo."
    />
    @if (store.notice(); as notice) {
      <p
        class="note-box mb-4"
        [class.orange]="notice.kind === 'error'"
        [attr.role]="notice.kind === 'error' ? 'alert' : 'status'"
      >
        <span>{{ notice.text }}</span>
        <button type="button" class="btn" (click)="store.dismissNotice()">Entendido</button>
      </p>
    }
    @if (store.status() === 'loading') {
      <p class="empty" role="status">Cargando la configuración…</p>
    } @else if (store.sinAcceso()) {
      <p class="note-box orange" role="status">
        No tienes permiso para ver la configuración del restaurante. Pídele acceso a quien
        administra tu equipo.
      </p>
    } @else {
      @if (store.puedeVerNegocio()) {
        <app-negocio-section />
      }
      @if (store.puedeVerSedes()) {
        <app-sedes-section />
      }
      @if (store.puedeVerSalon()) {
        <app-salon-section />
      }
    }
  `,
})
export class ConfiguracionReal implements OnInit {
  readonly store = inject(ConfiguracionStore);

  ngOnInit(): void {
    void this.store.load();
  }
}

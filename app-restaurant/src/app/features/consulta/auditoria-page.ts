import { ChangeDetectionStrategy, Component, computed, inject, signal } from '@angular/core';
import { RestaurantStore } from '../../core/demo/restaurant-store';
import { PageHeader } from '../../shared/ui/page-header';
export function csvCell(value: string): string {
  return '"' + value.replace(/^[\s]*[=+\-@]/, "'$&").replaceAll('"', '""') + '"';
}
@Component({
  imports: [PageHeader],
  changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<app-page-header
      title="Cada cambio tiene una historia"
      description="Consulta quién hizo qué y cuándo. Un historial que da contexto."
      eyebrow="AUDITORÍA"
      ><button class="btn" (click)="download()">Descargar CSV</button></app-page-header
    >
    <p class="note-box mb-6">
      Historial de solo lectura en la aplicación. La persistencia local de demostración no es una
      auditoría inmutable de producción.
    </p>
    <div class="section-toolbar">
      <label class="search"
        ><input
          aria-label="Buscar evento de auditoría"
          placeholder="Buscar usuario, entidad o detalle..."
          (input)="query.set($any($event.target).value); page.set(1)" /></label
      ><span class="subtle">{{ filtered().length }} eventos</span>
    </div>
    <div class="panel table-scroll">
      <table>
        <thead>
          <tr>
            <th>Momento</th>
            <th>Usuario</th>
            <th>Entidad / registro</th>
            <th>Acción</th>
            <th>Detalle</th>
          </tr>
        </thead>
        <tbody>
          @for (a of rows(); track a.id) {
            <tr>
              <td>{{ date(a.at) }}</td>
              <td>{{ a.actor }}</td>
              <td>
                {{ a.entity }}<small>{{ a.record }}</small>
              </td>
              <td>
                <span class="audit-action">{{ a.action }}</span>
              </td>
              <td class="audit-detail">{{ a.detail }}</td>
            </tr>
          } @empty {
            <tr>
              <td colspan="5"><div class="empty-state">No hay eventos que coincidan.</div></td>
            </tr>
          }
        </tbody>
      </table>
    </div>
    <nav aria-label="Paginación de auditoría" class="flex justify-end items-center gap-4 mt-5">
      <button class="btn" [disabled]="page() === 1" (click)="page.update(previous)">Anterior</button
      ><span class="subtle">Página {{ page() }} de {{ pages() }}</span
      ><button class="btn" [disabled]="page() >= pages()" (click)="page.update(next)">
        Siguiente
      </button>
    </nav>`,
})
export class AuditoriaPage {
  readonly store = inject(RestaurantStore);
  readonly query = signal('');
  readonly page = signal(1);
  readonly filtered = computed(() =>
    this.store
      .state()
      .audit.slice()
      .reverse()
      .filter((a) =>
        [a.actor, a.entity, a.record, a.action, a.detail]
          .join(' ')
          .toLowerCase()
          .includes(this.query().toLowerCase()),
      ),
  );
  readonly pages = computed(() => Math.max(1, Math.ceil(this.filtered().length / 20)));
  readonly rows = computed(() => this.filtered().slice((this.page() - 1) * 20, this.page() * 20));
  readonly previous = (p: number) => Math.max(1, p - 1);
  readonly next = (p: number) => p + 1;
  date(at: string): string {
    return new Date(at).toLocaleString('es-PE');
  }
  download(): void {
    const rows = this.filtered().map((a) =>
      [a.at, a.actor, a.entity, a.record, a.action, a.detail].map(csvCell).join(','),
    );
    const blob = new Blob(
      ['\uFEFFFecha,Usuario,Entidad,Registro,Acción,Detalle\r\n' + rows.join('\r\n')],
      { type: 'text/csv;charset=utf-8' },
    );
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a');
    anchor.href = url;
    anchor.download = 'mesa-auditoria.csv';
    anchor.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
}

import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core';
const labels: Record<string, string> = {
  NUEVO: 'Por enviar',
  PENDIENTE: 'Pendiente',
  REGISTRADO: 'Registrado',
  PREPARACION: 'En preparación',
  LISTO: 'Listo para servir',
  ENTREGADO: 'Entregado',
  CANCELADO: 'Cancelado',
  CANCELADA: 'Cancelada',
  ABIERTA: 'Abierta',
  CERRADA: 'Cerrada',
  CONFIRMADA: 'Confirmada',
  ATENDIDA: 'Atendida',
  NO_ASISTIO: 'No asistió',
  BORRADOR: 'Borrador',
  EN_RUTA: 'En ruta',
  SIMULADO: 'Simulado',
};
@Component({
  selector: 'app-status',
  template:
    '<span class="pill" [class.green]="green()" [class.orange]="orange()" [class.red]="red()"><i class="dot"></i>{{ label() }}</span>',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class StatusBadge {
  readonly value = input.required<string>();
  readonly label = computed(() => labels[this.value()] ?? this.value());
  readonly green = computed(() =>
    ['LISTO', 'CONFIRMADA', 'ABIERTA', 'ATENDIDA', 'Disponible'].includes(this.value()),
  );
  readonly orange = computed(() =>
    ['PREPARACION', 'PENDIENTE', 'BORRADOR', 'REGISTRADO', 'NUEVO'].includes(this.value()),
  );
  readonly red = computed(() =>
    ['CANCELADO', 'CANCELADA', 'NO_ASISTIO', 'Pausado'].includes(this.value()),
  );
}

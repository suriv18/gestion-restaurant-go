import { Component } from '@angular/core';
import { RouterLink } from '@angular/router';
@Component({
  imports: [RouterLink],
  template:
    '<div class="empty-state"><h1>Esta página no está en la carta</h1><p>Revisa la dirección o vuelve al salón.</p><a class="btn primary" routerLink="/salon">Ir al salón</a></div>',
})
export class NotFound {}

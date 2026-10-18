import { ChangeDetectionStrategy, Component, input } from '@angular/core';
@Component({
  selector: 'app-page-header',
  template:
    '<header class="page-head"><div><div class="eyebrow">{{ eyebrow() }}</div><h1>{{ title() }}</h1><p>{{ description() }}</p></div><div class="head-actions"><ng-content /></div></header>',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class PageHeader {
  readonly title = input.required<string>();
  readonly description = input('');
  readonly eyebrow = input('TU RESTAURANTE, EN UN SOLO LUGAR');
}

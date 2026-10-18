import { ChangeDetectionStrategy, Component, input } from '@angular/core';
import { Icon } from './icon';
@Component({
  selector: 'app-metric',
  imports: [Icon],
  template:
    '<article class="metric"><div><div class="metric-label">{{ label() }}</div><strong class="metric-value">{{ value() }}</strong><div class="metric-caption">{{ caption() }}</div></div><span class="metric-icon"><app-icon [name]="icon()" /></span></article>',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Metric {
  readonly label = input.required<string>();
  readonly value = input.required<string | number>();
  readonly caption = input('');
  readonly icon = input('table');
}

import { ChangeDetectionStrategy, Component, input } from '@angular/core';
const paths: Record<string, string> = {
  user: 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM4 21v-2c0-7 16-7 16 0v2',
  sun: 'M12 16a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM12 2v2M12 20v2M2 12h2M20 12h2m-15-7 2 2m10 10 2 2M5 19l2-2M17 7l2-2',
  moon: 'M20 15A9 9 0 0 1 9 4 9 9 0 1 0 20 15Z',
  monitor: 'M3 3h18v14H3zM12 17v4M8 21h8',
  palette:
    'M12 3a9 9 0 1 0 0 18h2c2 0 3-3 1-4-2-1-1-3 1-3h2c5 0 4-11-6-11ZM7 8h.01M12 6h.01M17 8h.01M6 13h.01',
  mail: 'M3 5h18v14H3zM3 5l9 8 9-8',
  lock: 'M5 10h14v11H5zM8 10V6a4 4 0 0 1 8 0v4M12 14v3',
  'eye-off':
    'm3 3 18 18M10 5h2c7 0 11 7 11 7s-1 2-3 4M6 6c-3 2-5 6-5 6s4 7 11 7c2 0 4-1 6-2M10 10a3 3 0 0 0 4 4',
  grid: 'M3 3h7v7H3zM14 3h7v7h-7zM3 14h7v7H3zM14 14h7v7h-7z',
  table: 'M4 8h16v8H4zM7 3h10M7 21h10M1 10v4M23 10v4',
  chef: 'M7 14v6h10v-6M7 16h10M7 14C1 14 1 6 7 6c1-5 9-5 10 0 6 0 6 8 0 8',
  calendar: 'M4 5h16v16H4zM8 3v4M16 3v4M4 10h16M8 14h2M14 14h2M8 17h2',
  bike: 'M5 20a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM19 20a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM5 16l5-8 5 8h4L15 5h3M8 8h5',
  wallet: 'M3 6h16v14H3zM3 6V4l14-2v4M15 11h7v5h-7zM18 13.5h.1',
  box: 'm3 7 9-5 9 5v10l-9 5-9-5V7Zm0 0 9 5 9-5M12 12v10M7 4.5l10 5',
  cart: 'M2 3h3l3 13h11l3-9H6M9 21h.01M18 21h.01M9 19a1 1 0 1 0 0 2 1 1 0 0 0 0-2ZM18 19a1 1 0 1 0 0 2 1 1 0 0 0 0-2Z',
  book: 'M12 5v16M12 5C9 2 5 2 2 3v16c4-1 7-1 10 2 3-3 6-3 10-2V3c-3-1-7-1-10 2Z',
  receipt: 'M5 2v20l3-2 4 2 4-2 3 2V2l-3 2-4-2-4 2-3-2ZM9 8h6M9 12h6M9 16h4',
  users:
    'M16 21v-3c0-5-12-5-12 0v3M10 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM17 4a4 4 0 0 1 0 7M20 21v-3c0-2-1-3-3-4',
  shield: 'm12 2 9 4v6c0 6-9 10-9 10S3 18 3 12V6l9-4ZM8 12l3 3 5-6',
  settings:
    'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8ZM9 2h6l1 3 3 1 3 5-2 2v3l-3 4h-3l-2 2-3-2H6l-4-4v-3l-1-2 3-5 3-1 2-3Z',
  clock: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20ZM12 6v6l4 2',
  plus: 'M12 5v14M5 12h14',
  chevron: 'm9 5 7 7-7 7',
  down: 'm6 9 6 6 6-6',
  arrow: 'M4 12h16m-6-6 6 6-6 6',
  close: 'm6 6 12 12M6 18 18 6',
  check: 'm5 12 4 4L19 6',
  search: 'M10 17a7 7 0 1 0 0-14 7 7 0 0 0 0 14Zm5-2 6 6',
  bell: 'M5 17h14l-2-4V8a5 5 0 0 0-10 0v5l-2 4ZM10 21h4',
  help: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20ZM9 8c0-4 7-3 6 1-.5 2-3 2-3 5M12 18h.01',
  store:
    'M4 10v11h16V10M2 10l3-7h14l3 7M2 10c0 4 5 4 5 0 0 4 5 4 5 0 0 4 5 4 5 0 0 4 5 4 5 0M9 21v-7h6v7',
  logout: 'M9 4H3v16h6M9 12h12m-5-5 5 5-5 5',
  menu: 'M4 6h16M4 12h16M4 18h16',
  edit: 'm15 4 5 5M3 21l5-1L21 7c2-2-2-6-4-4L4 16l-1 5Z',
  trash: 'M3 6h18M6 6l1 15h10l1-15M9 6V3h6v3M10 10v7M14 10v7',
  restore: 'M3 3v6h6M3 9c3-8 17-8 18 2 1 10-14 13-17 4',
  eye: 'M1 12s4-7 11-7 11 7 11 7-4 7-11 7S1 12 1 12ZM12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z',
  leaf: 'M4 20C0 9 10 3 21 3c0 11-6 20-17 17ZM4 20 17 7',
  drink: 'M6 3h12l-1 11c-1 5-9 5-10 0L6 3ZM12 17v5M8 22h8M7 8h10',
  plate:
    'M12 20a8 8 0 1 0 0-16 8 8 0 0 0 0 16ZM12 17a5 5 0 1 0 0-10 5 5 0 0 0 0 10ZM1 3v6M1 12v9M23 3v18',
  cake: 'M4 12h16v9H4zM3 12c0-4 4-4 6-1 2-3 4-3 6 0 2-3 6-3 6 1M8 8V5M16 8V5M8 2v.1M16 2v.1',
  pin: 'M12 22s8-8 8-14a8 8 0 0 0-16 0c0 6 8 14 8 14ZM12 11a3 3 0 1 0 0-6 3 3 0 0 0 0 6Z',
  send: 'm22 2-7 20-4-9L2 9 22 2ZM11 13 22 2',
  split: 'M5 3v6c0 3 14 3 14 6v6M5 21v-6c0-3 14-3 14-6V3m-3 3 3-3 3 3M2 18l3 3 3-3',
  info: 'M12 22a10 10 0 1 0 0-20 10 10 0 0 0 0 20ZM12 11v6M12 7h.01',
  download: 'M12 3v12m-5-5 5 5 5-5M4 16v5h16v-5',
  spark: 'm12 2 3 7 7 3-7 3-3 7-3-7-7-3 7-3 3-7',
  pause: 'M8 5v14M16 5v14',
  play: 'm7 3 14 9-14 9V3Z',
  phone: 'M5 3H2c-1 11 10 22 21 19v-4l-6-3-3 3c-3-2-6-5-7-8l3-2-3-5H5Z',
};
@Component({
  selector: 'app-icon',
  template:
    '<svg class="icon" aria-hidden="true" viewBox="0 0 24 24"><path [attr.d]="path()" /></svg>',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class Icon {
  readonly name = input('box');
  path(): string {
    return paths[this.name()] ?? paths['box'];
  }
}

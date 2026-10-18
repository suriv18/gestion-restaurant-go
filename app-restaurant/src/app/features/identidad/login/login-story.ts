import { ChangeDetectionStrategy, Component } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Icon } from '../../../shared/ui/icon';

@Component({
  selector: 'app-login-story',
  imports: [RouterLink, Icon],
  templateUrl: './login-story.html',
  styleUrl: './login-story.css',
  changeDetection: ChangeDetectionStrategy.OnPush,
})
export class LoginStory {}

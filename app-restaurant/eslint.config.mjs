import eslint from '@eslint/js';
import tseslint from 'typescript-eslint';
import angular from 'angular-eslint';
export default tseslint.config(
  { ignores:['dist/**','node_modules/**','coverage/**','playwright-report/**','test-results/**','src/app/core/demo/engine/**'] },
  { files:['src/**/*.ts'], extends:[eslint.configs.recommended,...tseslint.configs.recommended,...angular.configs.tsRecommended], processor:angular.processInlineTemplates },
  { files:['src/**/*.html'], extends:[...angular.configs.templateRecommended,...angular.configs.templateAccessibility] }
);

import { fileURLToPath } from 'node:url';

import { paraglideVitePlugin } from '@inlang/paraglide-js';
import { svelte, vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import Icons from 'unplugin-icons/vite';
import { defineConfig } from 'vite-plus';

export default defineConfig({
	root: fileURLToPath(new URL('.', import.meta.url)),
	publicDir: '../static',
	plugins: [
		paraglideVitePlugin({
			project: fileURLToPath(new URL('../project.inlang', import.meta.url)),
			outdir: fileURLToPath(new URL('../src/lib/paraglide', import.meta.url)),
			strategy: ['baseLocale']
		}),
		svelte({ configFile: false, preprocess: vitePreprocess() }),
		tailwindcss(),
		Icons({ compiler: 'svelte', autoInstall: false })
	],
	resolve: {
		alias: {
			'$app/env': fileURLToPath(new URL('./app-environment.ts', import.meta.url)),
			'$app/paths': fileURLToPath(new URL('./app-paths.ts', import.meta.url)),
			'#lib': fileURLToPath(new URL('../src/lib', import.meta.url))
		}
	},
	server: { host: '127.0.0.1', port: 4175, strictPort: true, fs: { allow: [fileURLToPath(new URL('../..', import.meta.url))] } }
});

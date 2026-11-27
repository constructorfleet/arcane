import { mount } from 'svelte';

import './layout.css';
import ResourceMenus from './resource-menus.svelte';

document.documentElement.classList.toggle('dark', matchMedia('(prefers-color-scheme: dark)').matches);
const target = document.getElementById('app');
if (!target) throw new Error('Missing conformance root');
mount(ResourceMenus, { target });

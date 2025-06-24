<script lang="ts">
import { Navbar, NavbarBrand, Container, Styles, Progress, ThemeToggler } from '@sveltestrap/sveltestrap';
import ThemeToggle from '$lib/components/ThemeToggle.svelte';
import { navigating } from '$app/stores';
import { writable } from 'svelte/store';
import { setContext } from 'svelte';
	import { M } from 'vitest/dist/chunks/reporters.d.BFLkQcL6.js';
	import { m } from '$lib/paraglide/messages.js';

// Store local para processamento com contador
const processingStore = writable({ activeTasks: 0 });

// Disponibiliza o store globalmente
setContext('processingStore', processingStore);

$: showProgress = $navigating || $processingStore.activeTasks > 0;
</script>

<ThemeToggler let:currentColorMode>
	<Progress 
		striped={showProgress}
		animated={showProgress} 
		value={100} 
		color={currentColorMode === 'light' ? 'dark' : 'light'}
		class="rounded-0"
	/>
</ThemeToggler>

<Styles/>
<Navbar>
    <NavbarBrand href="/">{m.app_name }</NavbarBrand>
    <ThemeToggle />
</Navbar>
<Container>
  <slot />
</Container>

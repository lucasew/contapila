<script lang="ts">
import { Navbar, NavbarBrand, Container, Styles, Progress } from '@sveltestrap/sveltestrap';
import ThemeToggle from '$lib/components/ThemeToggle.svelte';
import { navigating } from '$app/stores';
import { writable } from 'svelte/store';
import { setContext } from 'svelte';

// Store local para processamento com contador
const processingStore = writable({ activeTasks: 0 });

// Disponibiliza o store globalmente
setContext('processingStore', processingStore);

$: showProgress = $navigating || $processingStore.activeTasks > 0;
</script>

<Progress 
	striped={showProgress}
	animated={showProgress} 
	value={100} 
	style="border-radius: 0" 
/>

<Styles/>
<Navbar>
    <NavbarBrand href="/">Contapila</NavbarBrand>
    <ThemeToggle />
</Navbar>
<Container>
  <slot />
</Container>

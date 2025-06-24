<script lang="ts">
import { Navbar, NavbarBrand, Container, Styles, Progress } from '@sveltestrap/sveltestrap';
import ThemeToggle from '$lib/components/ThemeToggle.svelte';
import { navigating } from '$app/stores';
import { processingStore } from '$lib/stores';

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

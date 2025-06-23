<script lang="ts">
import { Navbar, NavbarBrand, Container, Styles } from '@sveltestrap/sveltestrap';
import ThemeToggle from '$lib/components/ThemeToggle.svelte';

</script>
<Styles/>
<Navbar>
    <NavbarBrand href="/">Contapila</NavbarBrand>
    <ThemeToggle />
</Navbar>
<Container>
  <slot />
</Container>

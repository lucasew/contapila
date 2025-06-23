<script lang="ts">
import { Navbar, NavbarBrand, Container } from '@sveltestrap/sveltestrap';
import ThemeToggle from '$lib/ThemeToggle.svelte';

</script>

<Navbar color="dark" dark class="mb-4">
  <Container class="d-flex justify-content-between align-items-center">
    <NavbarBrand href="/">Contapila</NavbarBrand>
    <ThemeToggle />
  </Container>
</Navbar>
<Container>
  <slot />
</Container>

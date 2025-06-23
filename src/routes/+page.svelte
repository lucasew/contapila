<script lang="ts">
	import {
		createCoreBeancountModule,
		createCustomReportingModule,
		createTransactionModule
	} from '$lib/beancount.js';
	import { createParser } from '$lib/parser.js';

	let files: FileList | undefined = $state();
	let content: any[] = $state([]);
	let erro: string | null = $state(null);

	const parser = createParser({
		modules: [
			createCoreBeancountModule(),
			createTransactionModule(),
			createCustomReportingModule()
		],
		fieldParsers: {},
		customValidators: {}
	});

	$effect(() => {
		console.log(files);
		if (!files) return;
		if (files.length != 1) return;
		files
			.item(0)
			?.text()
			.then((text: string) => {
				try {
					content = parser(text);
					erro = null;
				} catch (e: unknown) {
					erro = e instanceof Error ? e.message : String(e);
					content = [];
				}
			});
	});
</script>

<h1>Beancount preview</h1>

<input type="file" bind:files />
{#if erro != null}
	<p><b>Erro: </b>: {erro}</p>
{/if}

<pre>{JSON.stringify(content, null, 2)}</pre>

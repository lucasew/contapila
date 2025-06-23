<script lang="ts">
	import {
		createCoreBeancountModule,
		createCustomReportingModule,
		createTransactionModule
	} from '$lib/beancount.js';
	import { createParser } from '$lib/parser.js';
	import { Table} from '@sveltestrap/sveltestrap';

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
		console.log(content);
	});
</script>

<h1>Beancount preview</h1>

<input type="file" bind:files />
{#if erro != null}
	<p><b>Erro: </b>: {erro}</p>
{/if}

{#if content.length > 0}
	<Table striped bordered hover responsive>
		<thead>
			<tr>
				<th>Tipo</th>
				<th>Data</th>
				<th>Detalhes</th>
			</tr>
		</thead>
		<tbody>
			{#each content as entidade, i}
				<tr>
					<td>{entidade.kind}</td>
					<td>{entidade.date}</td>
					<td>
						<details>
							<summary>Ver detalhes</summary>
							<pre>{JSON.stringify(entidade, null, 2)}</pre>
						</details>
					</td>
				</tr>
			{/each}
		</tbody>
	</Table>
{:else}
	<p>Nenhuma entidade encontrada.</p>
{/if}

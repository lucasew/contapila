<script lang="ts">
	import {
		createCoreBeancountModule,
		createCustomReportingModule,
		createTransactionModule
	} from '$lib/beancount.js';
	import { createParser } from '$lib/parser.js';
	import { Table, Badge, Button, Collapse, ListGroup, ListGroupItem, Row, Col, Accordion, AccordionItem, Icon } from '@sveltestrap/sveltestrap';
	import PostingItem from '$lib/components/PostingItem.svelte';
	import TipoBadge from '$lib/components/TipoBadge.svelte';
	import FileUpload from '$lib/components/FileUpload.svelte';

	let files: FileList | undefined = $state();
	let content: any[] = $state([]);
	let erro: string | null = $state(null);
	let openCollapse: Record<number, boolean> = $state({});
	let openPostings: boolean[][] = [];

	const parser = createParser({
		modules: [
			createCoreBeancountModule(),
			createTransactionModule(),
			createCustomReportingModule()
		],
		fieldParsers: {},
		customValidators: {}
	});

	let linhasTabela = $derived(content.map(entidadeParaLinhaTabela));

	function getMainAccount(entidade: any) {
		if (entidade.kind === 'transaction' && entidade.postings && entidade.postings.length > 0) {
			return entidade.postings[0].account;
		}
		return entidade.account || '';
	}

	function getMainValue(entidade: any) {
		if (entidade.kind === 'transaction' && entidade.postings && entidade.postings.length > 0) {
			return entidade.postings[0].amount?.value;
		}
		return entidade.amount?.value;
	}

	function getMainCurrency(entidade: any) {
		if (entidade.kind === 'transaction' && entidade.postings && entidade.postings.length > 0) {
			return entidade.postings[0].amount?.currency;
		}
		return entidade.amount?.currency;
	}

	function getNarration(entidade: any) {
		return entidade.narration || entidade.comment || '';
	}

	function getTags(entidade: any) {
		return entidade.tags || [];
	}

	function getPostingType(posting: any, entidade: any) {
		// Para transações, o tipo pode ser flag ou keyword
		if (entidade.flag) return entidade.flag;
		if (entidade.keyword) return entidade.keyword;
		return '';
	}

	function toggleCollapse(idx: number) {
		openCollapse = { ...openCollapse, [idx]: !openCollapse[idx] };
	}

	function togglePosting(i: number, j: number) {
		// Garante nova referência para reatividade
		openPostings = openPostings.slice();
		openPostings[i] = (openPostings[i] || []).slice();
		openPostings[i][j] = !openPostings[i]?.[j];
	}

	function entidadeParaLinhaTabela(entidade: any) {
		let titulo = '';
		let subtitulo = '';
		let tags = [];
		let postings = undefined;
		let meta = entidade.meta;
		if (entidade.kind === 'balance') {
			titulo = entidade.account;
			subtitulo = entidade.amount ? `${entidade.amount.value} ${entidade.amount.currency}` : '';
		} else if (entidade.kind === 'open' || entidade.kind === 'close') {
			titulo = entidade.account;
			subtitulo = '';
		} else {
			if (entidade.payee) titulo = entidade.payee;
			if (entidade.narration) subtitulo = entidade.narration;
			else if (entidade.comment) subtitulo = entidade.comment;
		}
		if (entidade.tags) tags = entidade.tags;
		if (entidade.postings) postings = entidade.postings;
		return {
			data: entidade.date,
			tipo: entidade.kind,
			titulo,
			subtitulo,
			tags,
			postings,
			meta,
			narration: entidade.narration,
			comment: entidade.comment
		};
	}

	$effect(() => {
		// Garante que a reatividade funcione ao remover arquivos
		if (!files || files.length === 0) {
			content = [];
			erro = null;
			return;
		}
		console.log(files);
		if (files.length < 1) return;
		const allEntries: any[] = [];
		let errorFound: string | null = null;
		const promises = Array.from(files).map(file =>
			file.text().then(text => {
				try {
					// Cria um parser com o nome do arquivo como filename
					const parserWithFilename = createParser({
						modules: [
							createCoreBeancountModule(),
							createTransactionModule(),
							createCustomReportingModule()
						],
						fieldParsers: {},
						customValidators: {}
					}, file.name);
					const entries = parserWithFilename(text);
					allEntries.push(...entries);
				} catch (e: unknown) {
					errorFound = `Erro no arquivo ${file.name}: ` + (e instanceof Error ? e.message : String(e));
				}
			})
		);
		Promise.all(promises).then(() => {
			if (errorFound) {
				erro = errorFound;
				content = [];
			} else {
				erro = null;
				// Ordena por data crescente
				allEntries.sort((a, b) => (a.date || '').localeCompare(b.date || ''));
				content = allEntries;
			}
			console.log($inspect(content));
		});
	});
</script>

<FileUpload on:change={e => files = e.detail.files} />
{#if erro != null}
	<p><b>Erro: </b>: {erro}</p>
{/if}

{#if linhasTabela.length > 0}
	<Accordion>
		{#each linhasTabela as linha, i}
			<AccordionItem>
				<Row slot="header" class="align-items-center w-100">
					<Col class="col-auto text-nowrap" >{linha.data}</Col>
					<Col class="col-auto text-nowrap" >
						<TipoBadge tipo={linha.tipo} />
					</Col>
					<Col>
						{#if linha.titulo}
							<strong>{linha.titulo}</strong>
						{/if}
						{#if linha.subtitulo}
							{#if linha.titulo} {linha.subtitulo}{:else}{linha.subtitulo}{/if}
						{/if}
						{#each linha.tags as tag}
							<Badge color="secondary" class="me-1">{tag}</Badge>
						{/each}
					</Col>
				</Row>
				{#if linha.postings}
					{#each linha.postings as posting}
						<PostingItem posting={posting} />
					{/each}
				{/if}
				<details class="mt-2">
					<summary><strong>Atributos</strong></summary>
				{#if linha.meta}
					
						<ul class="ms-3 mb-2">
							{#each Object.entries(linha.meta) as [k, v]}
								<li><strong>{k}:</strong> <span class="ms-3">{JSON.stringify(v)}</span></li>
							{/each}
						</ul>
					
				{/if}
			</details>
			</AccordionItem>
		{/each}
	</Accordion>
{:else}
	<p>Nenhuma entidade encontrada.</p>
{/if}

<script lang="ts">
	import {
		createCoreBeancountModule,
		createCustomReportingModule,
		createTransactionModule
	} from '$lib/core/beancount.js';
	import { createParser } from '$lib/core/parser.js';
	import { Table, Badge, Button, Collapse, ListGroup, ListGroupItem, Row, Col, Accordion, AccordionItem, Icon } from '@sveltestrap/sveltestrap';
	import PostingItem from '$lib/components/PostingItem.svelte';
	import TipoBadge from '$lib/components/TipoBadge.svelte';
	import FileUpload from '$lib/components/FileUpload.svelte';
	import { getContext } from 'svelte';
	import type { Writable } from 'svelte/store';
	import { ParserWorker } from '$lib/workers/parser/wrapper.js';
	import { onDestroy } from 'svelte';
	import { m } from '$lib/paraglide/messages.js';

	let files: FileList | undefined = $state();
	let content: any[] = $state([]);
	let erro: string | null = $state(null);
	let openCollapse: Record<number, boolean> = $state({});
	let openPostings: boolean[][] = [];

	// Obtém o store do layout
	const processingStore = getContext<Writable<{ activeTasks: number }>>('processingStore');

	// Cria o worker do parser
	const parserWorker = new ParserWorker();

	// Cleanup quando o componente for destruído
	onDestroy(() => {
		parserWorker.terminate();
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
		let links = [];
		let postings = undefined;
		let meta = entidade.meta;
		if (entidade.kind === 'balance') {
			titulo = entidade.account;
			subtitulo = entidade.amount ? `${entidade.amount.value} ${entidade.amount.currency}` : '';
		} else if (entidade.kind === 'price') {
			titulo = entidade.commodity;
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
		if (entidade.links) links = entidade.links;
		if (entidade.postings) postings = entidade.postings;
		return {
			data: entidade.date,
			tipo: entidade.kind,
			titulo,
			subtitulo,
			tags,
			links,
			postings,
			meta,
			narration: entidade.narration,
			comment: entidade.comment
		};
	}

	$effect(() => {
		if (!files || files.length === 0) {
			content = [];
			erro = null;
			processingStore.update(state => ({ activeTasks: Math.max(0, state.activeTasks - 1) }));
			return;
		}
		console.log(files);
		if (files.length < 1) return;

		// Inicia o processamento
		processingStore.update(state => ({ activeTasks: state.activeTasks + 1 }));

		// Processa arquivos usando Web Worker
		(async () => {
			try {
				const fileArray = Array.from(files ?? []);
				const filesData = await Promise.all(
					fileArray.map(async (file) => ({
						text: await file.text(),
						filename: file.name
					}))
				);

				const results = await parserWorker.parseMultipleFiles(filesData);
				
				const allEntries: any[] = [];
				let errorFound: string | null = null;

				for (const result of results) {
					if (result.success) {
						allEntries.push(...result.entries);
					} else {
						errorFound = result.error;
						break;
					}
				}

				if (errorFound) {
					erro = errorFound;
					content = [];
				} else {
					erro = null;
					allEntries.sort((a, b) => (a.date || '').localeCompare(b.date || ''));
					content = allEntries;
				}
			} catch (error) {
				erro = `Erro no processamento: ${error instanceof Error ? error.message : String(error)}`;
				content = [];
			} finally {
				// Finaliza o processamento
				processingStore.update(state => ({ activeTasks: Math.max(0, state.activeTasks - 1) }));
			}
		})();
	});
</script>

<FileUpload on:change={e => files = e.detail.files} />

{#if erro != null}
	<p><b>{m.error_label()}:</b> {erro}</p>
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
						{#if linha.tipo === 'unknown_directive'}
							<strong>{linha.meta?.type}</strong>
						{:else if linha.titulo}
							<strong>{linha.titulo}</strong>
						{/if}
						{#if linha.subtitulo}
							{#if linha.titulo} {linha.subtitulo}{:else}{linha.subtitulo}{/if}
						{/if}
						{#each linha.tags as tag}
							<Badge color="secondary" class="me-1">{tag}</Badge>
						{/each}
						{#each linha.links as link}
							<Badge color="info" outline class="me-1">^{link}</Badge>
						{/each}
					</Col>
				</Row>
				{#if linha.postings}
					{#each linha.postings as posting}
						<PostingItem posting={posting} />
					{/each}
				{/if}
				<details class="mt-2">
					<summary><strong>{m.file_attributes_title()}</strong></summary>
					{#if linha.meta && Object.keys(linha.meta).length > 0}
						<ul class="ms-3 mb-2">
							{#each Object.entries(linha.meta) as [k, v]}
								<li><strong>{k}:</strong> <span class="ms-3">{JSON.stringify(v)}</span></li>
							{/each}
						</ul>
					{:else}
						<span class="ms-3 text-muted">{m.no_extra_attributes()}</span>
					{/if}
				</details>
			</AccordionItem>
		{/each}
	</Accordion>
{:else}
	<p>{m.no_entities_found()}</p>
{/if}
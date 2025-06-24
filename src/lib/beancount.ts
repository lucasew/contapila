// Reexport your entry components here
// Constantes de regex no topo

import {
	REGEX_PATTERNS,
	advanceCursor,
	createParser,
	createParserConfig,
	isAtEnd,
	parseAccount,
	parseAmount,
	parseDate,
	parseNewline,
	parseQuotedString,
	parseRegex,
	parseString,
	parseTagOrLink,
	parseTagsAndLinks,
	parseYAML,
	peekChar,
	skipWhitespace,
	type AmountValue,
	type BaseEntry,
	type DirectiveDefinition,
	type DirectiveModule,
	type FieldDefinition,
	sliceLineAtSemicolonOutsideString
} from './parser.js';

// Módulos funcionais
const createCoreBeancountModule = (): DirectiveModule => ({
	name: 'core-beancount',
	version: '1.0.0',
	directives: [
		{
			kind: 'open',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'open')
				},
				{ name: 'account', type: 'account', required: true },
				{
					name: 'currencies',
					type: 'array',
					required: false,
					parser: (cursor) => {
						const result = parseRegex(cursor, REGEX_PATTERNS.CURRENCY);
						if (!result) return null;
						let current = result.cursor;
						const currencies = [result.value[1]];

						// Parse additional comma-separated currencies
						while (peekChar(current) === ',') {
							current = advanceCursor(current, 1); // Skip comma
							current = skipWhitespace(current);
							const nextResult = parseRegex(current, REGEX_PATTERNS.CURRENCY);
							if (!nextResult) break;
							currencies.push(nextResult.value[1]);
							current = nextResult.cursor;
						}

						return { value: currencies, cursor: current };
					}
				}
			]
		},
		{
			kind: 'close',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'close')
				},
				{ name: 'account', type: 'account', required: true }
			]
		},
		{
			kind: 'balance',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'balance')
				},
				{ name: 'account', type: 'account', required: true },
				{ name: 'amount', type: 'amount', required: true }
			]
		},
		{
			kind: 'price',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'price')
				},
				{ name: 'commodity', type: 'string', required: true },
				{ name: 'amount', type: 'amount', required: true }
			]
		},
		{
			kind: 'note',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'note')
				},
				{ name: 'account', type: 'account', required: true },
				{ name: 'comment', type: 'string', required: true, parser: parseQuotedString }
			]
		}
	]
});

const createTransactionModule = (): DirectiveModule => ({
	name: 'transactions',
	version: '1.0.0',
	dependencies: ['core-beancount'],
	directives: [
		{
			kind: 'transaction',
			customParser: (cursor, fields) => {
				let current = cursor;

				const dateResult = parseDate(current);
				if (!dateResult) return null;
				current = skipWhitespace(dateResult.cursor);

				const flagResult = parseRegex(current, REGEX_PATTERNS.FLAG);
				if (!flagResult) return null;
				current = skipWhitespace(flagResult.cursor);

				let payee: string | undefined;
				let narration: string;

				const firstQuotedResult = parseQuotedString(current);
				if (!firstQuotedResult) return null;
				current = skipWhitespace(firstQuotedResult.cursor);

				const secondQuotedResult = parseQuotedString(current);
				if (secondQuotedResult) {
					// Two quoted strings: first is payee, second is narration
					payee = firstQuotedResult.value;
					narration = secondQuotedResult.value;
					current = skipWhitespace(secondQuotedResult.cursor);
				} else {
					// One quoted string: it's the narration, no payee
					narration = firstQuotedResult.value;
				}

				// Pule espaços e parseie tags/links ANTES de cortar no comentário
				current = skipWhitespace(current);
				let tags: string[] | undefined;
				let links: string[] | undefined;
				const tagsLinksResult = parseTagsAndLinks(current);
				if (tagsLinksResult) {
					tags = tagsLinksResult.value.filter(i => i.type === 'tag').map(i => i.value);
					links = tagsLinksResult.value.filter(i => i.type === 'link').map(i => i.value);
					current = tagsLinksResult.cursor;
				}
				current = skipWhitespace(current);

				// Agora corta o comentário inline (se houver) APENAS na linha principal
				const restOfLine = current.text.slice(current.position).split('\n')[0];
				let inString = false;
				let semicolonPos = -1;
				for (let i = 0; i < restOfLine.length; i++) {
					const char = restOfLine[i];
					if (char === '"') inString = !inString;
					if (char === ';' && !inString) {
						semicolonPos = i;
						break;
					}
				}
				// Verifica se há linhas indentadas após a linha principal
				const afterLine = current.text.slice(current.position + restOfLine.length);
				const hasIndented = /^\s+\S/m.test(afterLine);
				if (semicolonPos !== -1 && !hasIndented) {
					// Só finalize a transação se após o ; só houver comentário ou espaço até o fim da linha
					const afterSemicolon = restOfLine.slice(semicolonPos + 1);
					// Verifica se há tags/links após o comentário
					const hasTagsOrLinksAfterComment = /[#^]/.test(afterSemicolon);
					if ((/^\s*;.*$/.test(afterSemicolon) || /^\s*$/.test(afterSemicolon)) && !hasTagsOrLinksAfterComment) {
						current = advanceCursor(current, semicolonPos);
						while (!isAtEnd(current) && peekChar(current) !== '\n') {
							current = advanceCursor(current, 1);
						}
						if (!isAtEnd(current) && peekChar(current) === '\n') {
							current = advanceCursor(current, 1);
						}
						return {
							value: {
								kind: 'transaction',
								date: dateResult.value,
								flag: flagResult.value[1],
								payee,
								narration,
								postings: [],
								meta: {},
								tags: tags || [],
								links: links || []
							},
							cursor: current
						};
					}
				}

				// Pule linhas em branco após o header
				while (!isAtEnd(current)) {
					let c = peekChar(current);
					if (c === '\n') {
						current = advanceCursor(current, 1);
					} else if (c === ' ') {
						// só avança se for espaço no início da linha
						let temp = current.position;
						while (!isAtEnd(current) && peekChar({ ...current, position: temp }) === ' ') temp++;
						if (peekChar({ ...current, position: temp }) === '\n') {
							current = { ...current, position: temp + 1 };
						} else {
							break;
						}
					} else {
						break;
					}
				}
				// Agora processa todas as linhas indentadas
				let meta: any = {};
				const postings: any[] = [];
				while (!isAtEnd(current)) {
					// Pule linhas em branco
					let temp = current.position;
					while (!isAtEnd(current) && peekChar({ ...current, position: temp }) === ' ') temp++;
					if (peekChar({ ...current, position: temp }) === '\n') {
						current = { ...current, position: temp + 1 };
						continue;
					}
					// Verifica se a linha atual começa com pelo menos dois espaços
					let lineStart = current.position;
					let spaces = 0;
					while (!isAtEnd(current) && peekChar({ ...current, position: lineStart }) === ' ') {
						spaces++;
						lineStart++;
					}
					if (spaces < 2) break;
					current = { ...current, position: lineStart };
					// Check if this line is a comment
					const commentResult = parseRegex(current, REGEX_PATTERNS.COMMENT);
					if (commentResult) {
						const restOfLine = parseRegex(current, REGEX_PATTERNS.REST_OF_LINE);
						if (restOfLine) current = restOfLine.cursor;
						const commentNewline = parseNewline(current);
						if (commentNewline) current = commentNewline.cursor;
						continue;
					}
					// Check if this line is metadata (YAML key-value)
					const yamlKeyCheck = parseRegex(current, REGEX_PATTERNS.YAML_KEY);
					if (yamlKeyCheck) {
						const key = yamlKeyCheck.value[1];
						current = yamlKeyCheck.cursor;
						let value: any;
						const quotedStr = parseQuotedString(current);
						if (quotedStr) {
							value = quotedStr.value;
							current = quotedStr.cursor;
						} else {
							const valueMatch = parseRegex(current, REGEX_PATTERNS.YAML_VALUE);
							if (valueMatch) {
								const rawValue = valueMatch.value[1].trim();
								if (REGEX_PATTERNS.INTEGER.test(rawValue)) value = parseInt(rawValue);
								else if (REGEX_PATTERNS.FLOAT.test(rawValue)) value = parseFloat(rawValue);
								else if (rawValue === 'true' || rawValue === 'false') value = rawValue === 'true';
								else value = rawValue || null;
								current = valueMatch.cursor;
							}
						}
						meta[key] = value;
						const metaNewline = parseNewline(current);
						if (metaNewline) current = metaNewline.cursor;
					} else {
						// Parse as posting
						const accountResult = parseAccount(current);
						if (!accountResult) break;
						current = skipWhitespace(accountResult.cursor);
						let amount: AmountValue | undefined;
						const amountResult = parseAmount(current);
						if (amountResult) {
							amount = amountResult.value;
							current = amountResult.cursor;
						}
						const restOfLine = parseRegex(current, REGEX_PATTERNS.REST_OF_LINE);
						if (restOfLine) current = restOfLine.cursor;
						const postingNewline = parseNewline(current);
						if (postingNewline) current = postingNewline.cursor;
						let postingMeta: any;
						const postingMetaResult = parseYAML(current, 4);
						if (postingMetaResult) {
							postingMeta = postingMetaResult.value;
							current = postingMetaResult.cursor;
						}
						postings.push({
							account: accountResult.value,
							amount,
							meta: postingMeta || {}
						});
					}
				}
				// Depois, avance até a próxima linha que comece com data (diretiva) ou EOF
				while (!isAtEnd(current)) {
					let tempCursor = current;
					tempCursor = skipWhitespace(tempCursor);
					if (isAtEnd(tempCursor)) break;
					if (peekChar(tempCursor) === '\n') {
						current = advanceCursor(tempCursor, 1);
						continue;
					}
					const dateMatch = parseRegex(tempCursor, REGEX_PATTERNS.DATE);
					if (dateMatch) {
						current = tempCursor;
						break;
					}
					while (!isAtEnd(tempCursor) && peekChar(tempCursor) !== '\n') {
						tempCursor = advanceCursor(tempCursor, 1);
					}
					if (!isAtEnd(tempCursor) && peekChar(tempCursor) === '\n') {
						tempCursor = advanceCursor(tempCursor, 1);
					}
					current = tempCursor;
				}

				return {
					value: {
						kind: 'transaction',
						date: dateResult.value,
						flag: flagResult.value[1],
						payee,
						narration,
						postings,
						meta,
						tags: tags || [],
						links: links || []
					},
					cursor: current
				};
			},
			fields: []
		}
	]
});

const createCustomReportingModule = (): DirectiveModule => ({
	name: 'custom-reporting',
	version: '1.0.0',
	directives: [
		{
			kind: 'budget',
			fields: [
				{ name: 'date', type: 'date', required: true },
				{
					name: 'keyword',
					type: 'string',
					required: true,
					parser: (cursor) => parseString(cursor, 'budget')
				},
				{ name: 'account', type: 'account', required: true },
				{ name: 'amount', type: 'amount', required: true },
				{ name: 'period', type: 'string', required: false, defaultValue: 'monthly' }
			]
		}
	]
});

// Utilitários funcionais
const getAllDirectiveTypes = (modules: DirectiveModule[]): string[] =>
	modules.flatMap((module) => module.directives.map((directive) => directive.kind));

const getModuleByName = (modules: DirectiveModule[], name: string): DirectiveModule | undefined =>
	modules.find((module) => module.name === name);

// Utility: sum amounts by currency
function sumAmountsByCurrency(postings: { amount?: AmountValue }[]): Record<string, number> {
	const balance: Record<string, number> = {};
	for (const p of postings) {
		if (p.amount) {
			const { value, currency } = p.amount;
			balance[currency] = (balance[currency] || 0) + Number(value);
		}
	}
	return balance;
}

// Main validator/filler for transactions
function validateAndFillTransactions(entries: BaseEntry[]): { entries: BaseEntry[]; errors: { source: any; message: string; entry: BaseEntry }[] } {
	const errors: { source: any; message: string; entry: BaseEntry }[] = [];
	const newEntries = entries.map(entry => {
		if (entry.kind !== 'transaction') return entry;

		const postings = (entry.postings || []) as { amount?: AmountValue }[];
		const postingsWithoutAmount = postings.filter((p: { amount?: AmountValue }) => !p.amount);
		if (postingsWithoutAmount.length === 0) return entry; // Nothing to do

		if (postingsWithoutAmount.length === 1) {
			const balance = sumAmountsByCurrency(postings);
			const currencies = Object.keys(balance);
			if (currencies.length === 1) {
				const currency = currencies[0];
				const missingAmount = -balance[currency];
				const newPostings = postings.map((p: { amount?: AmountValue }) =>
					p.amount
						? p
						: { ...p, amount: { value: missingAmount, currency } }
				);
				return { ...entry, postings: newPostings };
			}
			// More than one currency: cannot infer
			errors.push({ source: entry.meta, message: 'Cannot infer missing amount: multiple currencies present.', entry });
			return entry;
		} else {
			// More than one posting without amount: error
			errors.push({ source: entry.meta, message: 'More than one posting without amount.', entry });
			return entry;
		}
	});
	return { entries: newEntries, errors };
}

// Exemplo de uso funcional
const exampleUsage = () => {
	const config = createParserConfig(
		[createCoreBeancountModule(), createTransactionModule(), createCustomReportingModule()],
		{
			// Custom field parser para percentual
			percentage: (cursor) => {
				const result = parseRegex(cursor, /^(\d+(?:\.\d+)?)%/);
				return result ? { value: parseFloat(result.value[1]) / 100, cursor: result.cursor } : null;
			}
		}
	);

	const parser = createParser(config);

	const testText = `
2024-01-01 open Assets:Cash USD,BRL #primary #checking #main
 description: "Conta principal"

2024-01-15 * "Mercado" "Compras do mês" #food #monthly #groceries
 category: "groceries"
 Assets:Cash      -200.00 BRL
 Expenses:Food     200.00 BRL

2024-02-01 budget Expenses:Food 500.00 BRL monthly #budget #food #monthly
 note: "Orçamento mensal para alimentação"

2024-01-01 price USD 5.25 BRL #exchange-rate #currency
2024-01-15 price AAPL 150.00 USD #stocks #tech #apple
`;

	const entries = parser(testText);
	const directiveTypes = getAllDirectiveTypes(config.modules);

	return { entries, directiveTypes };
};

export {
	createCoreBeancountModule,
	createCustomReportingModule,
	createTransactionModule,
	exampleUsage,
	getAllDirectiveTypes,
	getModuleByName,
	type AmountValue,
	type BaseEntry,
	type DirectiveDefinition,
	type DirectiveModule,
	type FieldDefinition,
	validateAndFillTransactions
};

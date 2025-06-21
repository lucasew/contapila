// Reexport your entry components here
// Constantes de regex no topo
const REGEX_PATTERNS = {
	WHITESPACE: /\s/,
	DATE: /^\d{4}-\d{2}-\d{2}/,
	AMOUNT: /^(-?\d+(?:\.\d+)?)\s+([A-Z][A-Z0-9_]+)(?:\s+\{[^}]+\})?/,
	ACCOUNT: /^([A-Z][A-Za-z0-9:_-]*)/,
	FLAG: /^([*!])/,
	NUMBER: /^(-?\d+(?:\.\d+)?)/,
	BOOLEAN: /^(true|false|yes|no|1|0)/i,
	STRING_UNQUOTED: /^(\S+)/,
	YAML_KEY: /^([a-zA-Z_][a-zA-Z0-9_-]*)\s*:\s+/,
	YAML_VALUE: /^([^\n]*)/,
	INTEGER: /^\d+$/,
	FLOAT: /^\d*\.\d+$/,
	COMMENT: /^;/,
	INDENT_TWO: /^  /,
	INDENT_FOUR: /^    /,
	CURRENCIES: /^([A-Z]{3}(?:,[A-Z]{3})*)/,
	REST_OF_LINE: /^[^\n]*/,
	EMAIL: /^([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/,
	TAG: /^#([a-zA-Z0-9_-]+)/
} as const;

// Tipos base
interface BaseEntry {
	kind: string;
	date: string;
	meta?: any;
	[key: string]: any;
}

interface ParseCursor {
	text: string;
	position: number;
	line: number;
	column: number;
}

interface ParseResult<T> {
	value: T;
	cursor: ParseCursor;
}

interface AmountValue {
	value: number;
	currency: string;
}

interface FieldDefinition {
	name: string;
	type: string;
	required: boolean;
	defaultValue?: any;
	validator?: (value: any) => boolean;
	parser?: FieldParser;
}

interface DirectiveDefinition {
	kind: string;
	keyword?: string;
	fields: FieldDefinition[];
	customParser?: DirectiveParser;
}

interface DirectiveModule {
	name: string;
	version: string;
	directives: DirectiveDefinition[];
	dependencies?: string[];
}

interface ParserConfig {
	modules: DirectiveModule[];
	fieldParsers: Record<string, FieldParser>;
	customValidators?: Record<string, (value: any) => boolean>;
}

// Tipos funcionais
type FieldParser = (cursor: ParseCursor) => ParseResult<any> | null;
type DirectiveParser = (
	cursor: ParseCursor,
	fields: FieldDefinition[]
) => ParseResult<BaseEntry> | null;
type ModuleValidator = (modules: DirectiveModule[]) => string[];

// Utilitários do cursor (funcionais)
const createCursor = (text: string): ParseCursor => ({
	text,
	position: 0,
	line: 1,
	column: 1
});

const advanceCursor = (cursor: ParseCursor, chars: number): ParseCursor => {
	let newPosition = cursor.position + chars;
	let newLine = cursor.line;
	let newColumn = cursor.column;

	for (let i = cursor.position; i < newPosition; i++) {
		if (cursor.text[i] === '\n') {
			newLine++;
			newColumn = 1;
		} else {
			newColumn++;
		}
	}

	return {
		...cursor,
		position: newPosition,
		line: newLine,
		column: newColumn
	};
};

const peekChar = (cursor: ParseCursor, offset = 0): string =>
	cursor.text[cursor.position + offset] || '';

const peekString = (cursor: ParseCursor, length: number): string =>
	cursor.text.slice(cursor.position, cursor.position + length);

const isAtEnd = (cursor: ParseCursor): boolean =>
	!cursor || !cursor.text || cursor.position >= cursor.text.length;

// Parsers básicos funcionais
const parseWhitespace = (cursor: ParseCursor): ParseCursor => {
	let current = cursor;
	while (
		!isAtEnd(current) &&
		REGEX_PATTERNS.WHITESPACE.test(peekChar(current)) &&
		peekChar(current) !== '\n'
	) {
		current = advanceCursor(current, 1);
	}
	return current;
};

const parseNewline = (cursor: ParseCursor): ParseResult<string> | null =>
	peekChar(cursor) === '\n' ? { value: '\n', cursor: advanceCursor(cursor, 1) } : null;

const parseRegex = (cursor: ParseCursor, regex: RegExp): ParseResult<RegExpMatchArray> | null => {
	const remaining = cursor.text.slice(cursor.position);
	const match = remaining.match(regex);

	return match && match.index === 0
		? { value: match, cursor: advanceCursor(cursor, match[0].length) }
		: null;
};

const parseString = (cursor: ParseCursor, str: string): ParseResult<string> | null =>
	peekString(cursor, str.length) === str
		? { value: str, cursor: advanceCursor(cursor, str.length) }
		: null;

const parseQuotedString = (cursor: ParseCursor): ParseResult<string> | null => {
	if (peekChar(cursor) !== '"') return null;

	let current = advanceCursor(cursor, 1);
	let value = '';

	while (!isAtEnd(current) && peekChar(current) !== '"') {
		if (peekChar(current) === '\\') {
			current = advanceCursor(current, 1);
			if (!isAtEnd(current)) {
				const escaped = peekChar(current);
				value += escaped === 'n' ? '\n' : escaped === 't' ? '\t' : escaped;
				current = advanceCursor(current, 1);
			}
		} else {
			value += peekChar(current);
			current = advanceCursor(current, 1);
		}
	}

	if (peekChar(current) === '"') {
		current = advanceCursor(current, 1);
		return { value, cursor: current };
	}

	throw new Error(`Unterminated string at line ${cursor.line}`);
};

// Parsers tipados funcionais
const parseDate = (cursor: ParseCursor): ParseResult<string> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.DATE);
	return result ? { value: result.value[0], cursor: result.cursor } : null;
};

const parseAmount = (cursor: ParseCursor): ParseResult<AmountValue> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.AMOUNT);
	return result
		? {
				value: {
					value: parseFloat(result.value[1]),
					currency: result.value[2]
				},
				cursor: result.cursor
			}
		: null;
};

const parseAccount = (cursor: ParseCursor): ParseResult<string> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.ACCOUNT);
	return result ? { value: result.value[1], cursor: result.cursor } : null;
};

const parseNumber = (cursor: ParseCursor): ParseResult<number> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.NUMBER);
	return result ? { value: parseFloat(result.value[1]), cursor: result.cursor } : null;
};

const parseBoolean = (cursor: ParseCursor): ParseResult<boolean> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.BOOLEAN);
	if (result) {
		const val = result.value[1].toLowerCase();
		return {
			value: val === 'true' || val === 'yes' || val === '1',
			cursor: result.cursor
		};
	}
	return null;
};

const parseStringField = (cursor: ParseCursor): ParseResult<string> | null => {
	const quoted = parseQuotedString(cursor);
	if (quoted) return quoted;

	const unquoted = parseRegex(cursor, REGEX_PATTERNS.STRING_UNQUOTED);
	return unquoted ? { value: unquoted.value[1], cursor: unquoted.cursor } : null;
};

const parseArray = (cursor: ParseCursor): ParseResult<string[]> | null => {
	const values: string[] = [];
	let current = cursor;

	while (!isAtEnd(current)) {
		const result = parseStringField(current);
		if (!result) break;

		values.push(result.value);
		current = parseWhitespace(result.cursor);

		const comma = parseString(current, ',');
		if (comma) {
			current = parseWhitespace(comma.cursor);
		} else {
			break;
		}
	}

	return values.length > 0 ? { value: values, cursor: current } : null;
};

const parseEmail = (cursor: ParseCursor): ParseResult<string> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.EMAIL);
	return result ? { value: result.value[1], cursor: result.cursor } : null;
};

const parseTag = (cursor: ParseCursor): ParseResult<string> | null => {
	if (!cursor || !cursor.text || isAtEnd(cursor)) {
		return null;
	}
	const result = parseRegex(cursor, REGEX_PATTERNS.TAG);
	return result ? { value: result.value[1], cursor: result.cursor } : null;
};

const parseTags = (cursor: ParseCursor): ParseResult<string[]> | null => {
	if (!cursor || !cursor.text || isAtEnd(cursor)) {
		return null;
	}

	const tags: string[] = [];
	let currentCursor = cursor;

	while (!isAtEnd(currentCursor)) {
		// Skip whitespace
		currentCursor = parseWhitespace(currentCursor);

		// Check if we're at end after whitespace
		if (isAtEnd(currentCursor)) {
			break;
		}

		// Try to parse a tag
		const tagResult = parseTag(currentCursor);
		if (tagResult) {
			tags.push(tagResult.value);
			currentCursor = tagResult.cursor;
		} else {
			break;
		}
	}

	return tags.length > 0 ? { value: tags, cursor: currentCursor } : null;
};

// Parser YAML funcional
const parseYAML = (cursor: ParseCursor, indentLevel: number): ParseResult<any> | null => {
	const baseIndent = ' '.repeat(indentLevel);
	const result: any = {};
	let current = cursor;

	const parseYAMLValue = (rawValue: string): any => {
		if (REGEX_PATTERNS.INTEGER.test(rawValue)) return parseInt(rawValue);
		if (REGEX_PATTERNS.FLOAT.test(rawValue)) return parseFloat(rawValue);
		if (rawValue === 'true' || rawValue === 'false') return rawValue === 'true';
		return rawValue || null;
	};

	while (!isAtEnd(current)) {
		if (peekChar(current) === '\n') {
			current = advanceCursor(current, 1);
			continue;
		}

		const indent = parseRegex(current, new RegExp(`^${baseIndent}\\s*`));
		if (!indent) break;

		current = indent.cursor;

		const keyMatch = parseRegex(current, REGEX_PATTERNS.YAML_KEY);
		if (!keyMatch) break;

		const key = keyMatch.value[1];
		current = keyMatch.cursor;

		let value: any;

		const quotedStr = parseQuotedString(current);
		if (quotedStr) {
			value = quotedStr.value;
			current = quotedStr.cursor;
		} else {
			const valueMatch = parseRegex(current, REGEX_PATTERNS.YAML_VALUE);
			if (valueMatch) {
				const rawValue = valueMatch.value[1].trim();
				value = parseYAMLValue(rawValue);
				current = valueMatch.cursor;
			}
		}

		result[key] = value;

		const newline = parseNewline(current);
		if (newline) {
			current = newline.cursor;
		}
	}

	return Object.keys(result).length > 0 ? { value: result, cursor: current } : null;
};

// Sistema de field parsers funcionais
const createBuiltinFieldParsers = (): Record<string, FieldParser> => ({
	string: parseStringField,
	number: parseNumber,
	boolean: parseBoolean,
	date: parseDate,
	amount: parseAmount,
	account: parseAccount,
	array: parseArray,
	email: parseEmail,
	object: (cursor) => parseYAML(cursor, 0),
	tag: parseTag,
	tags: parseTags
});

// Validação de módulos funcional
const validateModuleDependencies: ModuleValidator = (modules) => {
	const errors: string[] = [];
	const moduleNames = new Set(modules.map((m) => m.name));

	modules.forEach((module) => {
		if (module.dependencies) {
			module.dependencies.forEach((dep) => {
				if (!moduleNames.has(dep)) {
					errors.push(`Module '${module.name}' depends on missing module '${dep}'`);
				}
			});
		}
	});

	return errors;
};

// Parser de campo funcional
const parseField = (
	cursor: ParseCursor,
	field: FieldDefinition,
	fieldParsers: Record<string, FieldParser>
): ParseResult<any> | null => {
	if (field.parser) {
		return field.parser(cursor);
	}

	const parser = fieldParsers[field.type];
	if (!parser) {
		throw new Error(`No parser registered for field type: ${field.type}`);
	}

	const result = parser(cursor);

	if (result && field.validator && !field.validator(result.value)) {
		return null;
	}

	return result;
};

// Parser de diretiva funcional
const parseDirective = (
	cursor: ParseCursor,
	definition: DirectiveDefinition,
	fieldParsers: Record<string, FieldParser>
): ParseResult<BaseEntry> | null => {
	if (definition.customParser) {
		return definition.customParser(cursor, definition.fields);
	}

	let current = cursor;
	const entry: any = { kind: definition.kind };

	for (const field of definition.fields) {
		current = parseWhitespace(current);

		const result = parseField(current, field, fieldParsers);

		if (result) {
			entry[field.name] = result.value;
			current = result.cursor;
		} else if (field.required) {
			return null;
		} else if (field.defaultValue !== undefined) {
			entry[field.name] = field.defaultValue;
		}
	}

	current = parseWhitespace(current);

	// Parse tags at the end of the line
	const tagsResult = parseTags(current);
	if (tagsResult) {
		entry.tags = tagsResult.value;
		current = tagsResult.cursor;
	}

	current = parseWhitespace(current);
	const newlineResult = parseNewline(current);
	if (newlineResult) {
		current = newlineResult.cursor;
		const metaResult = parseYAML(current, 2);
		if (metaResult) {
			entry.meta = metaResult.value;
			current = metaResult.cursor;
		}
	}

	return { value: entry as BaseEntry, cursor: current };
};

// Parser principal funcional
const createParser = (config: ParserConfig) => {
	const validationErrors = validateModuleDependencies(config.modules);
	if (validationErrors.length > 0) {
		throw new Error(`Module validation failed: ${validationErrors.join(', ')}`);
	}

	const allDirectives = config.modules.flatMap((module) => module.directives);
	const fieldParsers = { ...createBuiltinFieldParsers(), ...config.fieldParsers };

	return (text: string): BaseEntry[] => {
		const entries: BaseEntry[] = [];
		let cursor = createCursor(text);

		while (!isAtEnd(cursor)) {
			cursor = parseWhitespace(cursor);

			if (peekChar(cursor) === '\n') {
				cursor = advanceCursor(cursor, 1);
				continue;
			}

			if (REGEX_PATTERNS.COMMENT.test(peekChar(cursor))) {
				while (!isAtEnd(cursor) && peekChar(cursor) !== '\n') {
					cursor = advanceCursor(cursor, 1);
				}
				continue;
			}

			if (isAtEnd(cursor)) break;

			let parsed = false;

			for (const definition of allDirectives) {
				const result = parseDirective(cursor, definition, fieldParsers);
				if (result) {
					entries.push(result.value);
					cursor = result.cursor;
					parsed = true;
					break;
				}
			}

			if (!parsed) {
				throw new Error(`Unknown directive at line ${cursor.line}, column ${cursor.column}`);
			}
		}

		return entries;
	};
};

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
						const result = parseRegex(cursor, REGEX_PATTERNS.CURRENCIES);
						if (!result) return null;
						const currencies = result.value[1].split(',');
						return { value: currencies, cursor: result.cursor };
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
				current = parseWhitespace(dateResult.cursor);

				const flagResult = parseRegex(current, REGEX_PATTERNS.FLAG);
				if (!flagResult) return null;
				current = parseWhitespace(flagResult.cursor);

				let payee: string | undefined;
				let narration: string;

				const firstQuotedResult = parseQuotedString(current);
				if (!firstQuotedResult) return null;
				current = parseWhitespace(firstQuotedResult.cursor);

				const secondQuotedResult = parseQuotedString(current);
				if (secondQuotedResult) {
					// Two quoted strings: first is payee, second is narration
					payee = firstQuotedResult.value;
					narration = secondQuotedResult.value;
					current = parseWhitespace(secondQuotedResult.cursor);
				} else {
					// One quoted string: it's the narration, no payee
					narration = firstQuotedResult.value;
				}

				// Parse tags at the end of the line
				let tags: string[] | undefined;
				const tagsResult = parseTags(current);
				if (tagsResult) {
					tags = tagsResult.value;
					current = tagsResult.cursor;
				}

				const newlineResult = parseNewline(current);
				if (newlineResult) {
					current = newlineResult.cursor;
				}

				// Parse transaction content (metadata and postings)
				let meta: any = {};
				const postings: any[] = [];

				while (!isAtEnd(current)) {
					const indentResult = parseRegex(current, REGEX_PATTERNS.INDENT_TWO);
					if (!indentResult) break;
					current = indentResult.cursor;

					// Check if this line is a comment
					const commentResult = parseRegex(current, REGEX_PATTERNS.COMMENT);
					if (commentResult) {
						// Skip comment line
						const restOfLine = parseRegex(current, REGEX_PATTERNS.REST_OF_LINE);
						if (restOfLine) {
							current = restOfLine.cursor;
						}
						const commentNewline = parseNewline(current);
						if (commentNewline) {
							current = commentNewline.cursor;
						}
						continue;
					}

					// Check if this line is metadata (YAML key-value)
					const yamlKeyCheck = parseRegex(current, REGEX_PATTERNS.YAML_KEY);
					if (yamlKeyCheck) {
						// Parse as metadata
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
						if (metaNewline) {
							current = metaNewline.cursor;
						}
					} else {
						// Parse as posting
						const accountResult = parseAccount(current);
						if (!accountResult) break;
						current = parseWhitespace(accountResult.cursor);

						let amount: AmountValue | undefined;
						const amountResult = parseAmount(current);
						if (amountResult) {
							amount = amountResult.value;
							current = amountResult.cursor;
						}

						const restOfLine = parseRegex(current, REGEX_PATTERNS.REST_OF_LINE);
						if (restOfLine) {
							current = restOfLine.cursor;
						}

						const postingNewline = parseNewline(current);
						if (postingNewline) {
							current = postingNewline.cursor;
						}

						let postingMeta: any;
						const postingMetaResult = parseYAML(current, 4);
						if (postingMetaResult) {
							postingMeta = postingMetaResult.value;
							current = postingMetaResult.cursor;
						}

						postings.push({
							account: accountResult.value,
							amount,
							meta: postingMeta
						});
					}
				}

				return {
					value: {
						kind: 'transaction',
						date: dateResult.value,
						flag: flagResult.value[1],
						payee,
						narration,
						postings,
						meta: Object.keys(meta).length > 0 ? meta : undefined,
						tags
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

// Função de conversão funcional
const convertBeancountToGeneralizedFormat = (
	beancountEntries: any[],
	config: ParserConfig
): BaseEntry[] => {
	const convertEntry = (entry: any): BaseEntry => {
		const converted: BaseEntry = {
			kind: entry.type,
			date: entry.date,
			meta: entry.meta
		};

		Object.entries(entry).forEach(([key, value]) => {
			if (key !== 'type' && key !== 'date' && key !== 'meta') {
				(converted as any)[key] = value;
			}
		});

		if (entry.type === 'transaction' && entry.postings) {
			converted.postings = entry.postings.map((posting: any) => ({
				account: posting.account,
				amount: posting.amount
					? {
							value: posting.amount.number,
							currency: posting.amount.currency
						}
					: undefined,
				cost: posting.cost
					? {
							value: posting.cost.number,
							currency: posting.cost.currency
						}
					: undefined,
				price: posting.price
					? {
							value: posting.price.number,
							currency: posting.price.currency
						}
					: undefined,
				flag: posting.flag,
				meta: posting.meta
			}));
		}

		if (entry.amount && entry.amount.number !== undefined) {
			(converted as any).amount = {
				value: entry.amount.number,
				currency: entry.amount.currency
			};
		}

		return converted;
	};

	return beancountEntries.map(convertEntry);
};

// Utilitários funcionais
const getAllDirectiveTypes = (modules: DirectiveModule[]): string[] =>
	modules.flatMap((module) => module.directives.map((directive) => directive.kind));

const getModuleByName = (modules: DirectiveModule[], name: string): DirectiveModule | undefined =>
	modules.find((module) => module.name === name);

const createParserConfig = (
	modules: DirectiveModule[],
	customFieldParsers: Record<string, FieldParser> = {},
	customValidators: Record<string, (value: any) => boolean> = {}
): ParserConfig => ({
	modules,
	fieldParsers: customFieldParsers,
	customValidators
});

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
	createParser,
	createParserConfig,
	createCoreBeancountModule,
	createTransactionModule,
	createCustomReportingModule,
	convertBeancountToGeneralizedFormat,
	getAllDirectiveTypes,
	getModuleByName,
	validateModuleDependencies,
	exampleUsage,
	parseTag,
	parseTags,
	type DirectiveModule,
	type DirectiveDefinition,
	type FieldDefinition,
	type ParserConfig,
	type BaseEntry,
	type AmountValue,
	type FieldParser,
	type DirectiveParser
};

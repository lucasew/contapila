export const REGEX_PATTERNS = {
	WHITESPACE: /\s/,
	DATE: /^\d{4}-\d{2}-\d{2}/,
	ACCOUNT: /^([A-Z][A-Za-z0-9:_-]+)/,
	FLAG: /^([*!])/,
	NUMBER: /^(-?\d+(?:\.\d+)?)/,
	BOOLEAN: /^(true|false|yes|no|1|0)/i,
	STRING_UNQUOTED: /^(\S+)/,
	YAML_KEY: /^([a-zA-Z_][a-zA-Z0-9_-]*)\s*:\s+/,
	YAML_VALUE: /^([^\n]*)/,
	INTEGER: /^\d+$/,
	FLOAT: /^\d*\.\d+$/,
	COMMENT: /^;/,
	COMMENT_INLINE: /;;/,
	INDENT_TWO: /^ {2}/,
	INDENT_FOUR: /^ {4}/,
	CURRENCY: /^([A-Z0-9_]+)/,
	REST_OF_LINE: /^[^\n]*/,
	EMAIL: /^([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/,
	TAG: /^#([a-zA-Z0-9_-]+)/,
	LINK: /^\^([a-zA-Z0-9_-]+)/
} as const;

export interface ParseCursor {
	text: string;
	position: number;
	line: number;
	column: number;
}

export interface ParseResult<T> {
	value: T;
	cursor: ParseCursor;
}

export interface DirectiveModule {
	name: string;
	version: string;
	directives: DirectiveDefinition[];
	dependencies?: string[];
}

export interface DirectiveDefinition {
	kind: string;
	keyword?: string;
	fields: FieldDefinition[];
	customParser?: DirectiveParser;
}

export interface BaseEntry {
	kind: string;
	date: string;
	meta?: any;
	[key: string]: any;
}

export interface AmountValue {
	value: number;
	currency: string;
}

export interface FieldDefinition {
	name: string;
	type: string;
	required: boolean;
	defaultValue?: any;
	validator?: (value: any) => boolean;
	parser?: FieldParser;
}

export interface ParserConfig {
	modules: DirectiveModule[];
	fieldParsers: Record<string, FieldParser>;
	customValidators?: Record<string, (value: any) => boolean>;
}

export type FieldParser = (cursor: ParseCursor) => ParseResult<any> | null;
export type DirectiveParser = (
	cursor: ParseCursor,
	fields: FieldDefinition[]
) => ParseResult<BaseEntry> | null;

export type CursorTransformer = (cursor: ParseCursor) => ParseCursor;

export const skipToEndOfLine: CursorTransformer = (cursor) => {
	let current = cursor;
	while (!isAtEnd(current) && peekChar(current) !== '\n') {
		current = advanceCursor(current, 1);
	}
	return current;
};

export const skipComments: CursorTransformer = (cursor) => {
	// Comentário de linha inteira (começa com ;)
	if (REGEX_PATTERNS.COMMENT.test(peekChar(cursor))) {
		return skipToEndOfLine(cursor);
	}
	return cursor;
};

export const skipInlineComments: CursorTransformer = (cursor) => {
	// Procura por comentários inline (;;) na linha atual
	let current = cursor;
	
	// Encontra o fim da linha atual
	while (!isAtEnd(current) && peekChar(current) !== '\n') {
		current = advanceCursor(current, 1);
	}
	
	const lineEnd = current.position;
	current = cursor;
	
	// Procura por ;; na linha atual
	while (current.position < lineEnd) {
		const remaining = cursor.text.slice(current.position, lineEnd);
		const match = remaining.match(REGEX_PATTERNS.COMMENT_INLINE);
		
		if (match && match.index === 0) {
			// Encontrou ;; - retorna o cursor na posição do ;;
			return current;
		}
		
		current = advanceCursor(current, 1);
	}
	
	return cursor;
};

export const skipAllComments: CursorTransformer = (cursor) => {
	// Primeiro verifica se é um comentário de linha inteira
	if (REGEX_PATTERNS.COMMENT.test(peekChar(cursor))) {
		return skipToEndOfLine(cursor);
	}
	
	// Depois verifica se há comentários inline na linha
	const inlineCommentPos = skipInlineComments(cursor);
	if (inlineCommentPos.position !== cursor.position) {
		// Encontrou comentário inline - pula até o fim da linha
		return skipToEndOfLine(inlineCommentPos);
	}
	
	return cursor;
};

export const createCursor = (text: string): ParseCursor => ({
	text,
	position: 0,
	line: 1,
	column: 1
});

export const advanceCursor = (cursor: ParseCursor, chars: number): ParseCursor => {
	const newPosition = cursor.position + chars;
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

export const peekChar = (cursor: ParseCursor, offset = 0): string =>
	cursor.text[cursor.position + offset] || '';

export const peekString = (cursor: ParseCursor, length: number): string =>
	cursor.text.slice(cursor.position, cursor.position + length);

export const isAtEnd = (cursor: ParseCursor): boolean =>
	!cursor || !cursor.text || cursor.position >= cursor.text.length;

export const parseWhitespace: CursorTransformer = (cursor) => {
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

export const skipWhitespace: CursorTransformer = parseWhitespace;

export const parseNewline = (cursor: ParseCursor): ParseResult<string> | null =>
	peekChar(cursor) === '\n' ? { value: '\n', cursor: advanceCursor(cursor, 1) } : null;

export const parseRegex = (
	cursor: ParseCursor,
	regex: RegExp
): ParseResult<RegExpMatchArray> | null => {
	const remaining = cursor.text.slice(cursor.position);
	const match = remaining.match(regex);

	return match && match.index === 0
		? { value: match, cursor: advanceCursor(cursor, match[0].length) }
		: null;
};

export const parseString = (cursor: ParseCursor, str: string): ParseResult<string> | null =>
	peekString(cursor, str.length) === str
		? { value: str, cursor: advanceCursor(cursor, str.length) }
		: null;

export const parseQuotedString = (cursor: ParseCursor): ParseResult<string> | null => {
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

export const parseDate = (cursor: ParseCursor): ParseResult<string> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.DATE);
	return result ? { value: result.value[0], cursor: result.cursor } : null;
};

export const parseCurrency = (cursor: ParseCursor): ParseResult<string> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.CURRENCY);
	return result ? { value: result.value[1], cursor: result.cursor } : null;
};

export const parseAmount = (cursor: ParseCursor): ParseResult<AmountValue> | null => {
	const numberResult = parseNumber(cursor);
	if (!numberResult) return null;

	const current = skipWhitespace(numberResult.cursor);
	const currencyResult = parseCurrency(current);
	if (!currencyResult) return null;

	return {
		value: {
			value: numberResult.value,
			currency: currencyResult.value
		},
		cursor: currencyResult.cursor
	};
};

export const parseAccount = (cursor: ParseCursor): ParseResult<string> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.ACCOUNT);
	return result ? { value: result.value[1], cursor: result.cursor } : null;
};

export const parseNumber = (cursor: ParseCursor): ParseResult<number> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.NUMBER);
	return result ? { value: parseFloat(result.value[1]), cursor: result.cursor } : null;
};

export const parseBoolean = (cursor: ParseCursor): ParseResult<boolean> | null => {
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

export const parseStringField = (cursor: ParseCursor): ParseResult<string> | null => {
	const quoted = parseQuotedString(cursor);
	if (quoted) return quoted;

	const unquoted = parseRegex(cursor, REGEX_PATTERNS.STRING_UNQUOTED);
	return unquoted ? { value: unquoted.value[1], cursor: unquoted.cursor } : null;
};

export const parseArray = (cursor: ParseCursor): ParseResult<string[]> | null => {
	const values: string[] = [];
	let current = cursor;

	while (!isAtEnd(current)) {
		const result = parseStringField(current);
		if (!result) break;

		values.push(result.value);
		current = skipWhitespace(result.cursor);

		const comma = parseString(current, ',');
		if (comma) {
			current = skipWhitespace(comma.cursor);
		} else {
			break;
		}
	}

	return values.length > 0 ? { value: values, cursor: current } : null;
};

export const parseEmail = (cursor: ParseCursor): ParseResult<string> | null => {
	const result = parseRegex(cursor, REGEX_PATTERNS.EMAIL);
	return result ? { value: result.value[1], cursor: result.cursor } : null;
};

export const parseTagOrLink = (cursor: ParseCursor): ParseResult<{type: 'tag'|'link', value: string}> | null => {
	if (!cursor || !cursor.text || isAtEnd(cursor)) {
		return null;
	}
	const tagResult = parseRegex(cursor, REGEX_PATTERNS.TAG);
	if (tagResult) {
		return { value: { type: 'tag', value: tagResult.value[1] }, cursor: tagResult.cursor };
	}
	const linkResult = parseRegex(cursor, REGEX_PATTERNS.LINK);
	if (linkResult) {
		return { value: { type: 'link', value: linkResult.value[1] }, cursor: linkResult.cursor };
	}
	return null;
};

export const parseTagsAndLinks = (cursor: ParseCursor): ParseResult<{type: 'tag'|'link', value: string}[]> | null => {
	if (!cursor || !cursor.text || isAtEnd(cursor)) {
		return null;
	}

	const items: {type: 'tag'|'link', value: string}[] = [];
	let currentCursor = cursor;

	while (!isAtEnd(currentCursor)) {
		currentCursor = skipWhitespace(currentCursor);
		if (isAtEnd(currentCursor)) {
			break;
		}

		const itemResult = parseTagOrLink(currentCursor);
		if (itemResult) {
			items.push(itemResult.value);
			currentCursor = itemResult.cursor;
		} else {
			break;
		}
	}

	return items.length > 0 ? { value: items, cursor: currentCursor } : null;
};

export const parseYAML = (cursor: ParseCursor, indentLevel: number): ParseResult<any> | null => {
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

export const createBuiltinFieldParsers = (): Record<string, FieldParser> => ({
	string: parseStringField,
	number: parseNumber,
	boolean: parseBoolean,
	date: parseDate,
	amount: parseAmount,
	account: parseAccount,
	array: parseArray,
	email: parseEmail,
	object: (cursor) => parseYAML(cursor, 0),
	tag: parseTagOrLink,
	tags: parseTagsAndLinks,
	tagsAndLinks: parseTagsAndLinks
});

export const parseField = (
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

// Função utilitária para cortar a linha no primeiro ; que não esteja dentro de aspas
export function sliceLineAtSemicolonOutsideString(cursor: ParseCursor): ParseCursor {
	const text = cursor.text.slice(cursor.position);
	let inString = false;
	for (let i = 0; i < text.length; i++) {
		const char = text[i];
		if (char === '"') {
			inString = !inString;
		} else if (char === ';' && !inString) {
			return advanceCursor(cursor, i);
		} else if (char === '\n') {
			break;
		}
	}
	return cursor;
}

export const parseDirective = (
	cursor: ParseCursor,
	definition: DirectiveDefinition,
	fieldParsers: Record<string, FieldParser>,
	filename: string = 'stdin'
): ParseResult<BaseEntry> | null => {
	if (definition.customParser) {
		return definition.customParser(cursor, definition.fields);
	}

	let current = cursor;
	const entry: any = { kind: definition.kind };
	const startLine = current.line;
	const startColumn = current.column;

	for (const field of definition.fields) {
		current = skipWhitespace(current);
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

	// Corta a linha no ponto e vírgula (se houver)
	const afterSemicolon = sliceLineAtSemicolonOutsideString(current);
	current = afterSemicolon;

	// Pule espaços e parseie tags/links
	current = skipWhitespace(current);
	const tagsLinksResult = parseTagsAndLinks(current);
	if (tagsLinksResult) {
		entry.tags = tagsLinksResult.value.filter(i => i.type === 'tag').map(i => i.value);
		entry.links = tagsLinksResult.value.filter(i => i.type === 'link').map(i => i.value);
		current = tagsLinksResult.cursor;
	}
	current = skipWhitespace(current);

	// Agora avance até o final da linha e consuma o \n (se houver)
	while (!isAtEnd(current) && peekChar(current) !== '\n') {
		current = advanceCursor(current, 1);
	}
	if (!isAtEnd(current) && peekChar(current) === '\n') {
		current = advanceCursor(current, 1);
	}

	// Agora chame parseYAML para consumir linhas indentadas como metadados
	let metaResult = parseYAML(current, 2);
	if (metaResult) {
		entry.meta = metaResult.value;
		current = metaResult.cursor;
		while (!isAtEnd(current)) {
			let posBefore = current.position;
			current = skipWhitespace(current);
			if (peekChar(current) === '\n') {
				current = advanceCursor(current, 1);
			} else if (current.position === posBefore) {
				break;
			}
		}
	} else {
		// Caso não haja metadados, consuma um \n se houver
		const newlineResult = parseNewline(current);
		if (newlineResult) {
			current = newlineResult.cursor;
		}
	}

	if (!entry.meta) entry.meta = {};
	entry.meta.location = `${filename}:${startLine}`;

	return { value: entry as BaseEntry, cursor: current };
};

export const createParser = (config: ParserConfig, filename: string = 'stdin') => {
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
			cursor = skipWhitespace(cursor);

			if (peekChar(cursor) === '\n') {
				cursor = advanceCursor(cursor, 1);
				continue;
			}

			// Só pula linhas que começam com ; ou ****
			if (peekChar(cursor) === ';' || (peekChar(cursor) === '*' && peekChar(cursor, 1) === '*' && peekChar(cursor, 2) === '*' && peekChar(cursor, 3) === '*')) {
				cursor = skipToEndOfLine(cursor);
				continue;
			}

			if (isAtEnd(cursor)) break;

			let parsed = false;

			for (const definition of allDirectives) {
				const result = parseDirective(cursor, definition, fieldParsers, filename);
				if (result) {
					entries.push(result.value);
					cursor = result.cursor;
					parsed = true;
					break;
				}
			}

			if (!parsed) {
				// Se a linha começa com uma data, verifique se há linhas indentadas após ela
				const dateMatch = parseRegex(cursor, REGEX_PATTERNS.DATE);
				if (dateMatch) {
					// Avança até o fim da linha
					let tempCursor = cursor;
					while (!isAtEnd(tempCursor) && peekChar(tempCursor) !== '\n') {
						tempCursor = advanceCursor(tempCursor, 1);
					}
					if (peekChar(tempCursor) === '\n') tempCursor = advanceCursor(tempCursor, 1);
					// Checa se próxima linha é indentada
					const fc = peekChar(tempCursor);
					const sc = peekChar(tempCursor, 1);
					const tc = peekChar(tempCursor, 2);
					const frc = peekChar(tempCursor, 3);
					if ((fc === ' ' && sc === ' ') || (fc === ' ' && sc === ' ' && tc === ' ' && frc === ' ')) {
						// Ignora bloco (linha + metadados)
						cursor = tempCursor;
						while (!isAtEnd(cursor)) {
							const fc2 = peekChar(cursor);
							const sc2 = peekChar(cursor, 1);
							const tc2 = peekChar(cursor, 2);
							const frc2 = peekChar(cursor, 3);
							if ((fc2 === ' ' && sc2 === ' ') || (fc2 === ' ' && sc2 === ' ' && tc2 === ' ' && frc2 === ' ')) {
								while (!isAtEnd(cursor) && peekChar(cursor) !== '\n') {
									cursor = advanceCursor(cursor, 1);
								}
								if (peekChar(cursor) === '\n') cursor = advanceCursor(cursor, 1);
							} else {
								break;
							}
						}
						continue;
					}
					// Se não há linhas indentadas, volta o cursor para a posição original para gerar unknown_directive normalmente
				}
				// Agrega linhas indentadas como parte do body da unknown_directive
				const start = cursor.position;
				let end = start;
				while (!isAtEnd(cursor) && peekChar(cursor) !== '\n') {
					cursor = advanceCursor(cursor, 1);
					end = cursor.position;
				}
				// Agregar linhas indentadas seguintes
				let nextLineStart = cursor.position;
				while (!isAtEnd(cursor)) {
					// Checa se próxima linha é indentada
					let isIndented = false;
					if (peekChar(cursor) === '\n') {
						let lookahead = cursor.position + 1;
						let spaces = 0;
						while (peekChar({ ...cursor, position: lookahead }) === ' ') {
							spaces++;
							lookahead++;
						}
						if (spaces >= 2) {
							isIndented = true;
						}
					}
					if (isIndented) {
						// Pula o \n
						cursor = advanceCursor(cursor, 1);
						let lineStart = cursor.position;
						while (!isAtEnd(cursor) && peekChar(cursor) !== '\n') {
							cursor = advanceCursor(cursor, 1);
						}
						end = cursor.position;
					} else {
						break;
					}
				}
				const bodyRaw = text.slice(start, end);
				// Remove indentação de cada linha agregada (inclusive a primeira)
				const body = bodyRaw.replace(/^ +/gm, '').trim();
				const bodyWords = body.split(/\s+/);
				let type = '';
				let date = '';
				if (bodyWords.length > 0) {
					if (REGEX_PATTERNS.DATE.test(bodyWords[0])) {
						date = bodyWords[0];
						type = bodyWords[1] || '';
					} else {
						type = bodyWords[0];
					}
				}
				let value = undefined;
				// Se houver pelo menos três "palavras" (data, tipo, value entre aspas), extrai o value
				const quotedMatches = body.match(/"([^"]*)"/g);
				if (quotedMatches && quotedMatches.length >= 2) {
					value = quotedMatches[1].replace(/\"/g, '').replace(/"/g, '');
				}
				entries.push({
					kind: 'unknown_directive',
					date,
					type,
					value,
					body,
					meta: {
						warning: `Unknown directive at line ${cursor.line}, column ${cursor.column}`,
						location: `${filename}:${cursor.line}`,
						type,
						body
					}
				});
				if (peekChar(cursor) === '\n') cursor = advanceCursor(cursor, 1);
			}
		}

		return entries;
	};
};

export const convertBeancountToGeneralizedFormat = (
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

export const createParserConfig = (
	modules: DirectiveModule[],
	customFieldParsers: Record<string, FieldParser> = {},
	customValidators: Record<string, (value: any) => boolean> = {}
): ParserConfig => ({
	modules,
	fieldParsers: customFieldParsers,
	customValidators
});

export type ModuleValidator = (modules: DirectiveModule[]) => string[];
export const validateModuleDependencies: ModuleValidator = (modules) => {
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

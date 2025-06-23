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
	INDENT_TWO: /^ {2}/,
	INDENT_FOUR: /^ {4}/,
	CURRENCY: /^([A-Z0-9_]+)/,
	REST_OF_LINE: /^[^\n]*/,
	EMAIL: /^([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})/,
	TAG: /^#([a-zA-Z0-9_-]+)/
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
	if (REGEX_PATTERNS.COMMENT.test(peekChar(cursor))) {
		return skipToEndOfLine(cursor);
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

export const parseTag = (cursor: ParseCursor): ParseResult<string> | null => {
	if (!cursor || !cursor.text || isAtEnd(cursor)) {
		return null;
	}
	const result = parseRegex(cursor, REGEX_PATTERNS.TAG);
	return result ? { value: result.value[1], cursor: result.cursor } : null;
};

export const parseTags = (cursor: ParseCursor): ParseResult<string[]> | null => {
	if (!cursor || !cursor.text || isAtEnd(cursor)) {
		return null;
	}

	const tags: string[] = [];
	let currentCursor = cursor;

	while (!isAtEnd(currentCursor)) {
		currentCursor = skipWhitespace(currentCursor);
		if (isAtEnd(currentCursor)) {
			break;
		}

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
	tag: parseTag,
	tags: parseTags
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

	// Após parsear os campos, se encontrar ';;', ignora o resto da linha
	current = skipWhitespace(current);
	const restOfLine = current.text.slice(current.position).split('\n')[0];
	const semicolonIndex = restOfLine.indexOf(';;');
	if (semicolonIndex !== -1) {
		current = advanceCursor(current, semicolonIndex);
		while (!isAtEnd(current) && peekChar(current) !== '\n') {
			current = advanceCursor(current, 1);
		}
	}

	current = skipWhitespace(current);

	const tagsResult = parseTags(current);
	if (tagsResult) {
		entry.tags = tagsResult.value;
		current = tagsResult.cursor;
	}

	current = skipWhitespace(current);
	const newlineResult = parseNewline(current);
	if (newlineResult) {
		current = newlineResult.cursor;
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

			if (REGEX_PATTERNS.COMMENT.test(peekChar(cursor))) {
				cursor = skipComments(cursor);
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
				const start = cursor.position;
				let end = start;
				while (!isAtEnd(cursor) && peekChar(cursor) !== '\n') {
					cursor = advanceCursor(cursor, 1);
					end = cursor.position;
				}
				const body = text.slice(start, end);
				const type = body.trim().split(/\s+/)[0] || '';
				entries.push({
					kind: 'unknown_directive',
					date: '',
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

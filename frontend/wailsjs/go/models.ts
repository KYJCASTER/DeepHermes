export namespace api {

	export class FunctionCall {
	    name: string;
	    arguments: string;

	    static createFrom(source: any = {}) {
	        return new FunctionCall(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.arguments = source["arguments"];
	    }
	}
	export class ToolCall {
	    index?: number;
	    id: string;
	    type: string;
	    function: FunctionCall;

	    static createFrom(source: any = {}) {
	        return new ToolCall(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.id = source["id"];
	        this.type = source["type"];
	        this.function = this.convertValues(source["function"], FunctionCall);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Message {
	    role: string;
	    content: string;
	    reasoning_content?: string;
	    tool_calls?: ToolCall[];
	    tool_call_id?: string;
	    name?: string;

	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	        this.reasoning_content = source["reasoning_content"];
	        this.tool_calls = this.convertValues(source["tool_calls"], ToolCall);
	        this.tool_call_id = source["tool_call_id"];
	        this.name = source["name"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace app {

	export class APIKeyTestRequest {
	    apiKey: string;
	    baseUrl: string;
	    model: string;
	    timeoutSeconds: number;
	    maxRetries: number;
	    proxyUrl: string;

	    static createFrom(source: any = {}) {
	        return new APIKeyTestRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.apiKey = source["apiKey"];
	        this.baseUrl = source["baseUrl"];
	        this.model = source["model"];
	        this.timeoutSeconds = source["timeoutSeconds"];
	        this.maxRetries = source["maxRetries"];
	        this.proxyUrl = source["proxyUrl"];
	    }
	}
	export class APIKeyTestResult {
	    ok: boolean;
	    message: string;
	    latencyMs: number;

	    static createFrom(source: any = {}) {
	        return new APIKeyTestResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.latencyMs = source["latencyMs"];
	    }
	}
	export class DiagnosticLog {
	    time: string;
	    level: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new DiagnosticLog(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	export class AppDiagnostics {
	    version: string;
	    buildCommit: string;
	    buildDate: string;
	    goVersion: string;
	    platform: string;
	    arch: string;
	    configPath: string;
	    dataDir: string;
	    sessionsDir: string;
	    portable: boolean;
	    minimizeToTray: boolean;
	    model: string;
	    mode: string;
	    baseUrl: string;
	    apiKeyStatus: string;
	    sessionCount: number;
	    memoryDir: string;
	    recentLogs: DiagnosticLog[];

	    static createFrom(source: any = {}) {
	        return new AppDiagnostics(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.buildCommit = source["buildCommit"];
	        this.buildDate = source["buildDate"];
	        this.goVersion = source["goVersion"];
	        this.platform = source["platform"];
	        this.arch = source["arch"];
	        this.configPath = source["configPath"];
	        this.dataDir = source["dataDir"];
	        this.sessionsDir = source["sessionsDir"];
	        this.portable = source["portable"];
	        this.minimizeToTray = source["minimizeToTray"];
	        this.model = source["model"];
	        this.mode = source["mode"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKeyStatus = source["apiKeyStatus"];
	        this.sessionCount = source["sessionCount"];
	        this.memoryDir = source["memoryDir"];
	        this.recentLogs = this.convertValues(source["recentLogs"], DiagnosticLog);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AppSettings {
	    model: string;
	    mode: string;
	    portable: boolean;
	    minimizeToTray: boolean;
	    maxTokens: number;
	    temperature: number;
	    baseUrl: string;
	    apiTimeout: number;
	    apiMaxRetries: number;
	    apiProxyUrl: string;
	    thinkingEnabled: boolean;
	    reasoningDisplay: string;
	    autoCowork: boolean;
	    toolMode: string;
	    toolOverrides: Record<string, string>;
	    bashBlocklist: string[];
	    initialPrompt: string;
	    roleCard: string;
	    worldBook: string;
	    ocrEnabled: boolean;
	    ocrProvider: string;
	    ocrBaseUrl: string;
	    ocrModel: string;
	    ocrPrompt: string;
	    ocrTimeout: number;

	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.mode = source["mode"];
	        this.portable = source["portable"];
	        this.minimizeToTray = source["minimizeToTray"];
	        this.maxTokens = source["maxTokens"];
	        this.temperature = source["temperature"];
	        this.baseUrl = source["baseUrl"];
	        this.apiTimeout = source["apiTimeout"];
	        this.apiMaxRetries = source["apiMaxRetries"];
	        this.apiProxyUrl = source["apiProxyUrl"];
	        this.thinkingEnabled = source["thinkingEnabled"];
	        this.reasoningDisplay = source["reasoningDisplay"];
	        this.autoCowork = source["autoCowork"];
	        this.toolMode = source["toolMode"];
	        this.toolOverrides = source["toolOverrides"];
	        this.bashBlocklist = source["bashBlocklist"];
	        this.initialPrompt = source["initialPrompt"];
	        this.roleCard = source["roleCard"];
	        this.worldBook = source["worldBook"];
	        this.ocrEnabled = source["ocrEnabled"];
	        this.ocrProvider = source["ocrProvider"];
	        this.ocrBaseUrl = source["ocrBaseUrl"];
	        this.ocrModel = source["ocrModel"];
	        this.ocrPrompt = source["ocrPrompt"];
	        this.ocrTimeout = source["ocrTimeout"];
	    }
	}
	export class BranchSessionRequest {
	    sessionId: string;
	    upToIndex: number;
	    nameSuffix: string;

	    static createFrom(source: any = {}) {
	        return new BranchSessionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.upToIndex = source["upToIndex"];
	        this.nameSuffix = source["nameSuffix"];
	    }
	}
	export class CharacterCardImportResult {
	    name: string;
	    roleCard: string;
	    worldBook: string;
	    source: string;

	    static createFrom(source: any = {}) {
	        return new CharacterCardImportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.roleCard = source["roleCard"];
	        this.worldBook = source["worldBook"];
	        this.source = source["source"];
	    }
	}
	export class ContextSummaryResult {
	    summary: string;
	    tokens: number;

	    static createFrom(source: any = {}) {
	        return new ContextSummaryResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.summary = source["summary"];
	        this.tokens = source["tokens"];
	    }
	}
	export class CreateSessionResult {
	    id: string;
	    name: string;
	    model: string;
	    createdAt: string;

	    static createFrom(source: any = {}) {
	        return new CreateSessionResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.model = source["model"];
	        this.createdAt = source["createdAt"];
	    }
	}

	export class ExportSessionRequest {
	    sessionId: string;
	    format: string;

	    static createFrom(source: any = {}) {
	        return new ExportSessionRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.format = source["format"];
	    }
	}
	export class FileEntry {
	    name: string;
	    path: string;
	    isDir: boolean;
	    size: number;
	    children?: FileEntry[];

	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.children = this.convertValues(source["children"], FileEntry);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FileSearchResult {
	    name: string;
	    path: string;
	    relativePath: string;
	    size: number;

	    static createFrom(source: any = {}) {
	        return new FileSearchResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.relativePath = source["relativePath"];
	        this.size = source["size"];
	    }
	}
	export class FileSnippet {
	    name: string;
	    path: string;
	    size: number;
	    content: string;
	    truncated: boolean;
	    binary: boolean;

	    static createFrom(source: any = {}) {
	        return new FileSnippet(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.content = source["content"];
	        this.truncated = source["truncated"];
	        this.binary = source["binary"];
	    }
	}
	export class MessageIndexRequest {
	    sessionId: string;
	    index: number;

	    static createFrom(source: any = {}) {
	        return new MessageIndexRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.index = source["index"];
	    }
	}
	export class OCRImageRequest {
	    fileName: string;
	    mimeType: string;
	    dataBase64: string;

	    static createFrom(source: any = {}) {
	        return new OCRImageRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.mimeType = source["mimeType"];
	        this.dataBase64 = source["dataBase64"];
	    }
	}
	export class OCRImageResult {
	    text: string;
	    provider: string;
	    model: string;
	    error?: string;

	    static createFrom(source: any = {}) {
	        return new OCRImageResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.error = source["error"];
	    }
	}
	export class OCRProviderPreset {
	    id: string;
	    name: string;
	    baseUrl: string;
	    model: string;

	    static createFrom(source: any = {}) {
	        return new OCRProviderPreset(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.model = source["model"];
	    }
	}
	export class TokenUsage {
	    promptTokens: number;
	    completionTokens: number;
	    totalTokens: number;
	    promptCacheHitTokens: number;
	    promptCacheMissTokens: number;
	    reasoningTokens: number;

	    static createFrom(source: any = {}) {
	        return new TokenUsage(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.promptTokens = source["promptTokens"];
	        this.completionTokens = source["completionTokens"];
	        this.totalTokens = source["totalTokens"];
	        this.promptCacheHitTokens = source["promptCacheHitTokens"];
	        this.promptCacheMissTokens = source["promptCacheMissTokens"];
	        this.reasoningTokens = source["reasoningTokens"];
	    }
	}
	export class RunMetrics {
	    usage: TokenUsage;
	    startedAt: string;
	    firstTokenAt?: string;
	    finishedAt: string;
	    firstTokenMs: number;
	    durationMs: number;
	    tokensPerSec: number;
	    finishReason?: string;
	    truncated?: boolean;

	    static createFrom(source: any = {}) {
	        return new RunMetrics(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.usage = this.convertValues(source["usage"], TokenUsage);
	        this.startedAt = source["startedAt"];
	        this.firstTokenAt = source["firstTokenAt"];
	        this.finishedAt = source["finishedAt"];
	        this.firstTokenMs = source["firstTokenMs"];
	        this.durationMs = source["durationMs"];
	        this.tokensPerSec = source["tokensPerSec"];
	        this.finishReason = source["finishReason"];
	        this.truncated = source["truncated"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SendMessageRequest {
	    sessionId: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new SendMessageRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.message = source["message"];
	    }
	}
	export class SessionInfo {
	    id: string;
	    name: string;
	    model: string;
	    createdAt: string;
	    updatedAt: string;
	    msgCount: number;
	    usage: TokenUsage;
	    lastRun?: RunMetrics;
	    contextSummaryTokens: number;

	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.model = source["model"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.msgCount = source["msgCount"];
	        this.usage = this.convertValues(source["usage"], TokenUsage);
	        this.lastRun = this.convertValues(source["lastRun"], RunMetrics);
	        this.contextSummaryTokens = source["contextSummaryTokens"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SessionStorageResult {
	    path: string;
	    sessions: number;

	    static createFrom(source: any = {}) {
	        return new SessionStorageResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.sessions = source["sessions"];
	    }
	}
	export class SpawnSubAgentRequest {
	    parentSessionId: string;
	    name: string;
	    agentType: string;
	    task: string;

	    static createFrom(source: any = {}) {
	        return new SpawnSubAgentRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.parentSessionId = source["parentSessionId"];
	        this.name = source["name"];
	        this.agentType = source["agentType"];
	        this.task = source["task"];
	    }
	}
	export class SubAgentStatus {
	    id: string;
	    name: string;
	    agentType: string;
	    status: string;
	    createdAt: string;
	    result: string;

	    static createFrom(source: any = {}) {
	        return new SubAgentStatus(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.agentType = source["agentType"];
	        this.status = source["status"];
	        this.createdAt = source["createdAt"];
	        this.result = source["result"];
	    }
	}

	export class ToolInfo {
	    name: string;
	    description: string;

	    static createFrom(source: any = {}) {
	        return new ToolInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	    }
	}
	export class ToolRollbackResult {
	    restored: boolean;
	    deleted: boolean;
	    path: string;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new ToolRollbackResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.restored = source["restored"];
	        this.deleted = source["deleted"];
	        this.path = source["path"];
	        this.message = source["message"];
	    }
	}
	export class UpdateContextSummaryRequest {
	    sessionId: string;
	    summary: string;

	    static createFrom(source: any = {}) {
	        return new UpdateContextSummaryRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.summary = source["summary"];
	    }
	}
	export class UpdateMessageRequest {
	    sessionId: string;
	    index: number;
	    content: string;

	    static createFrom(source: any = {}) {
	        return new UpdateMessageRequest(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.index = source["index"];
	        this.content = source["content"];
	    }
	}

}

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
	    index: number;
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
	    content?: string;
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
	
	export class AppSettings {
	    model: string;
	    maxTokens: number;
	    temperature: number;
	    baseUrl: string;
	    thinkingEnabled: boolean;
	    reasoningDisplay: string;
	    autoCowork: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.maxTokens = source["maxTokens"];
	        this.temperature = source["temperature"];
	        this.baseUrl = source["baseUrl"];
	        this.thinkingEnabled = source["thinkingEnabled"];
	        this.reasoningDisplay = source["reasoningDisplay"];
	        this.autoCowork = source["autoCowork"];
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
	export class SessionInfo {
	    id: string;
	    name: string;
	    model: string;
	    createdAt: string;
	    updatedAt: string;
	    msgCount: number;
	    usage: TokenUsage;
	    lastRun?: RunMetrics;
	
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

}

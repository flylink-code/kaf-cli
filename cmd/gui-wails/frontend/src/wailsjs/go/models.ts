export namespace main {

	export class UpdateInfo {
	    available: boolean;
	    current: string;
	    latest: string;
	    downloadURL: string;

	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.downloadURL = source["downloadURL"];
	    }
	}
	
	export class aiTasks {
	    structure: boolean;
	    typography: boolean;
	    noise: boolean;
	    metadata: boolean;
	
	    static createFrom(source: any = {}) {
	        return new aiTasks(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.structure = source["structure"];
	        this.typography = source["typography"];
	        this.noise = source["noise"];
	        this.metadata = source["metadata"];
	    }
	}
	export class aiConfig {
	    enabled: boolean;
	    base_url: string;
	    api_key: string;
	    model: string;
	    sample_chars: number;
	    tasks: aiTasks;
	
	    static createFrom(source: any = {}) {
	        return new aiConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.base_url = source["base_url"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	        this.sample_chars = source["sample_chars"];
	        this.tasks = this.convertValues(source["tasks"], aiTasks);
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
	
	export class aiTestResult {
	    ok: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new aiTestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	    }
	}
	export class convertAIRequest {
	    enabled: boolean;
	    structure: boolean;
	    typography: boolean;
	    noise: boolean;
	    metadata: boolean;
	    sampleChars: number;
	
	    static createFrom(source: any = {}) {
	        return new convertAIRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.structure = source["structure"];
	        this.typography = source["typography"];
	        this.noise = source["noise"];
	        this.metadata = source["metadata"];
	        this.sampleChars = source["sampleChars"];
	    }
	}
	export class convertRequest {
	    txtFile: string;
	    coverFile: string;
	    author: string;
	    format: string;
	    match: string;
	    volumeMatch: string;
	    dedup: boolean;
	    tips: boolean;
	    quotes: boolean;
	    ai: convertAIRequest;
	
	    static createFrom(source: any = {}) {
	        return new convertRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.txtFile = source["txtFile"];
	        this.coverFile = source["coverFile"];
	        this.author = source["author"];
	        this.format = source["format"];
	        this.match = source["match"];
	        this.volumeMatch = source["volumeMatch"];
	        this.dedup = source["dedup"];
	        this.tips = source["tips"];
	        this.quotes = source["quotes"];
	        this.ai = this.convertValues(source["ai"], convertAIRequest);
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
	export class guiConfig {
	    txt_file: string;
	    cover_file: string;
	    author: string;
	    format_index: number;
	    match: string;
	    volume_match: string;
	    dedup: boolean;
	    tips: boolean;
	    quotes: boolean;
	    ai: aiConfig;
	
	    static createFrom(source: any = {}) {
	        return new guiConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.txt_file = source["txt_file"];
	        this.cover_file = source["cover_file"];
	        this.author = source["author"];
	        this.format_index = source["format_index"];
	        this.match = source["match"];
	        this.volume_match = source["volume_match"];
	        this.dedup = source["dedup"];
	        this.tips = source["tips"];
	        this.quotes = source["quotes"];
	        this.ai = this.convertValues(source["ai"], aiConfig);
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
	export class sourceInsight {
	    bookname: string;
	    author: string;
	    cover: string;
	
	    static createFrom(source: any = {}) {
	        return new sourceInsight(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bookname = source["bookname"];
	        this.author = source["author"];
	        this.cover = source["cover"];
	    }
	}

}

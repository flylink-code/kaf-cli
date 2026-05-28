export namespace main {
	
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

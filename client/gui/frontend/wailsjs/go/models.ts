export namespace engine {
	
	export class StatusSnapshot {
	    state: number;
	    uptimeNs: number;
	    rttNs: number;
	    activeStreams: number;
	    bytesUp: number;
	    bytesDown: number;
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new StatusSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.state = source["state"];
	        this.uptimeNs = source["uptimeNs"];
	        this.rttNs = source["rttNs"];
	        this.activeStreams = source["activeStreams"];
	        this.bytesUp = source["bytesUp"];
	        this.bytesDown = source["bytesDown"];
	        this.lastError = source["lastError"];
	    }
	}

}

export namespace main {
	
	export class Settings {
	    nodeAddr: string;
	    socksPort: number;
	    secret: string;
	    autoStart: boolean;
	    startMinimized: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.nodeAddr = source["nodeAddr"];
	        this.socksPort = source["socksPort"];
	        this.secret = source["secret"];
	        this.autoStart = source["autoStart"];
	        this.startMinimized = source["startMinimized"];
	    }
	}

}


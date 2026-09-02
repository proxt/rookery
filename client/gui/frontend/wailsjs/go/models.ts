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
	
	export class CachedNode {
	    id: string;
	    name: string;
	    tags: string;
	    address: string;
	
	    static createFrom(source: any = {}) {
	        return new CachedNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.tags = source["tags"];
	        this.address = source["address"];
	    }
	}
	export class Subscription {
	    id: string;
	    name: string;
	    panelAddr: string;
	    token: string;
	    activeNodeId: string;
	    nodes: CachedNode[];
	
	    static createFrom(source: any = {}) {
	        return new Subscription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.panelAddr = source["panelAddr"];
	        this.token = source["token"];
	        this.activeNodeId = source["activeNodeId"];
	        this.nodes = this.convertValues(source["nodes"], CachedNode);
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
	    subscriptions: Subscription[];
	    activeSubscriptionId: string;
	    socksPort: number;
	    autoStart: boolean;
	    startMinimized: boolean;
	    systemWide: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.subscriptions = this.convertValues(source["subscriptions"], Subscription);
	        this.activeSubscriptionId = source["activeSubscriptionId"];
	        this.socksPort = source["socksPort"];
	        this.autoStart = source["autoStart"];
	        this.startMinimized = source["startMinimized"];
	        this.systemWide = source["systemWide"];
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
	
	
	export class UpdateInfo {
	    available: boolean;
	    currentVersion: string;
	    version: string;
	    notes: string;
	    downloadUrl: string;
	    size: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.currentVersion = source["currentVersion"];
	        this.version = source["version"];
	        this.notes = source["notes"];
	        this.downloadUrl = source["downloadUrl"];
	        this.size = source["size"];
	    }
	}

}


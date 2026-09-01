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
	
	export class Profile {
	    id: string;
	    name: string;
	    nodeAddr: string;
	    userId: string;
	    secret: string;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.nodeAddr = source["nodeAddr"];
	        this.userId = source["userId"];
	        this.secret = source["secret"];
	    }
	}
	export class AppSettings {
	    profiles: Profile[];
	    activeProfileId: string;
	    socksPort: number;
	    autoStart: boolean;
	    startMinimized: boolean;
	    systemWide: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profiles = this.convertValues(source["profiles"], Profile);
	        this.activeProfileId = source["activeProfileId"];
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

}


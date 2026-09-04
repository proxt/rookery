export namespace engine {
	
	export class StatusSnapshot {
	    state: number;
	    uptimeNs: number;
	    rttNs: number;
	    activeStreams: number;
	    bytesUp: number;
	    bytesDown: number;
	    lastError: string;
	    killSwitchEngaged: boolean;
	
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
	        this.killSwitchEngaged = source["killSwitchEngaged"];
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
	    routingRuleSet?: routing.RuleSet;
	
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
	        this.routingRuleSet = this.convertValues(source["routingRuleSet"], routing.RuleSet);
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
	    killSwitch: boolean;
	    subAutoRefreshMinutes: number;
	    subRefreshOnLaunch: boolean;
	    panelRoutingEnabled: boolean;
	    localRoutingRuleSets: routing.RuleSet[];
	
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
	        this.killSwitch = source["killSwitch"];
	        this.subAutoRefreshMinutes = source["subAutoRefreshMinutes"];
	        this.subRefreshOnLaunch = source["subRefreshOnLaunch"];
	        this.panelRoutingEnabled = source["panelRoutingEnabled"];
	        this.localRoutingRuleSets = this.convertValues(source["localRoutingRuleSets"], routing.RuleSet);
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
	
	export class RoutingExportResult {
	    path: string;
	    skipped: number;
	
	    static createFrom(source: any = {}) {
	        return new RoutingExportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.skipped = source["skipped"];
	    }
	}
	export class RoutingImportResult {
	    ruleSet: routing.RuleSet;
	    skipped: number;
	
	    static createFrom(source: any = {}) {
	        return new RoutingImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ruleSet = this.convertValues(source["ruleSet"], routing.RuleSet);
	        this.skipped = source["skipped"];
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

export namespace routing {
	
	export class Rule {
	    id: string;
	    type: string;
	    value: string;
	    action: string;
	
	    static createFrom(source: any = {}) {
	        return new Rule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.value = source["value"];
	        this.action = source["action"];
	    }
	}
	export class RuleSet {
	    id: string;
	    name: string;
	    rules: Rule[];
	
	    static createFrom(source: any = {}) {
	        return new RuleSet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.rules = this.convertValues(source["rules"], Rule);
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


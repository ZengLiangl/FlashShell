export namespace data {
	
	export class GlobalConfig {
	    appId: string;
	    windowsName: string;
	    configFile: string[];
	    lastOpenedFile: string;
	    workPaths: Record<string, string>;
	    machines?: define.Machine[];
	
	    static createFrom(source: any = {}) {
	        return new GlobalConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appId = source["appId"];
	        this.windowsName = source["windowsName"];
	        this.configFile = source["configFile"];
	        this.lastOpenedFile = source["lastOpenedFile"];
	        this.workPaths = source["workPaths"];
	        this.machines = this.convertValues(source["machines"], define.Machine);
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

export namespace define {
	
	export class Command {
	    name: string;
	    description: string;
	    type: string;
	    steps: string[];
	    machine?: string;
	    workdir?: string;
	
	    static createFrom(source: any = {}) {
	        return new Command(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.type = source["type"];
	        this.steps = source["steps"];
	        this.machine = source["machine"];
	        this.workdir = source["workdir"];
	    }
	}
	export class CommandStatus {
	    isRunning: boolean;
	    command: string;
	    output: string;
	
	    static createFrom(source: any = {}) {
	        return new CommandStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isRunning = source["isRunning"];
	        this.command = source["command"];
	        this.output = source["output"];
	    }
	}
	export class Machine {
	    encrypted_data?: string;
	    name: string;
	    keyfile?: string;
	
	    static createFrom(source: any = {}) {
	        return new Machine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.encrypted_data = source["encrypted_data"];
	        this.name = source["name"];
	        this.keyfile = source["keyfile"];
	    }
	}
	export class SubProject {
	    name: string;
	    description: string;
	    commands: Command[];
	
	    static createFrom(source: any = {}) {
	        return new SubProject(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.commands = this.convertValues(source["commands"], Command);
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
	export class Project {
	    name: string;
	    description: string;
	    workdir: string;
	    subprojects: SubProject[];
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.workdir = source["workdir"];
	        this.subprojects = this.convertValues(source["subprojects"], SubProject);
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
	export class Root {
	    projects: Project[];
	    machines: Machine[];
	
	    static createFrom(source: any = {}) {
	        return new Root(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projects = this.convertValues(source["projects"], Project);
	        this.machines = this.convertValues(source["machines"], Machine);
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
	
	export class SubProjectStatus {
	    projectName: string;
	    subProjectName: string;
	    isRunning: boolean;
	    currentCommand: string;
	    completedCommands: number;
	    totalCommands: number;
	    output: string;
	
	    static createFrom(source: any = {}) {
	        return new SubProjectStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectName = source["projectName"];
	        this.subProjectName = source["subProjectName"];
	        this.isRunning = source["isRunning"];
	        this.currentCommand = source["currentCommand"];
	        this.completedCommands = source["completedCommands"];
	        this.totalCommands = source["totalCommands"];
	        this.output = source["output"];
	    }
	}

}


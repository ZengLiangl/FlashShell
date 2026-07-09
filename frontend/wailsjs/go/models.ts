export namespace data {
	
	export class ThemeSettings {
	    mode: string;
	    terminalPreset: string;
	
	    static createFrom(source: any = {}) {
	        return new ThemeSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.terminalPreset = source["terminalPreset"];
	    }
	}
	export class LogSettings {
	    enabled: boolean;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new LogSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.path = source["path"];
	    }
	}
	export class GlobalConfig {
	    appId: string;
	    windowsName: string;
	    configFile: string[];
	    lastOpenedFile: string;
	    workPaths: Record<string, string>;
	    machines?: define.Machine[];
	    logSettings: LogSettings;
	    themeSettings: ThemeSettings;
	
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
	        this.logSettings = this.convertValues(source["logSettings"], LogSettings);
	        this.themeSettings = this.convertValues(source["themeSettings"], ThemeSettings);
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
	export class LogEntry {
	    fileName: string;
	    fullPath: string;
	    project: string;
	    subProject: string;
	    startedAt: string;
	    size: number;
	    success: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.fullPath = source["fullPath"];
	        this.project = source["project"];
	        this.subProject = source["subProject"];
	        this.startedAt = source["startedAt"];
	        this.size = source["size"];
	        this.success = source["success"];
	    }
	}
	
	export class SessionState {
	    sessionId: string;
	    lastOpenedFile: string;
	    theme: string;
	    terminalPreset: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.lastOpenedFile = source["lastOpenedFile"];
	        this.theme = source["theme"];
	        this.terminalPreset = source["terminalPreset"];
	    }
	}

}

export namespace define {
	
	export class Step {
	    command: string;
	    when?: string;
	    onFail?: string;
	    retry?: number;
	
	    static createFrom(source: any = {}) {
	        return new Step(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.command = source["command"];
	        this.when = source["when"];
	        this.onFail = source["onFail"];
	        this.retry = source["retry"];
	    }
	}
	export class Command {
	    name: string;
	    description: string;
	    type: string;
	    steps: Step[];
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
	        this.steps = this.convertValues(source["steps"], Step);
	        this.machine = source["machine"];
	        this.workdir = source["workdir"];
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
	    group?: string;
	    key_file?: string;
	
	    static createFrom(source: any = {}) {
	        return new Machine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.encrypted_data = source["encrypted_data"];
	        this.name = source["name"];
	        this.group = source["group"];
	        this.key_file = source["key_file"];
	    }
	}
	export class SubProject {
	    name: string;
	    description: string;
	    workdir?: string;
	    commands: Command[];
	
	    static createFrom(source: any = {}) {
	        return new SubProject(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.workdir = source["workdir"];
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
	export class SensitiveData {
	    host: string;
	    port: number;
	    user: string;
	    password?: string;
	
	    static createFrom(source: any = {}) {
	        return new SensitiveData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	    }
	}
	export class ShellStatus {
	    connected: boolean;
	    machineName: string;
	    host: string;
	    user: string;
	    isRunning: boolean;
	    currentCommand: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.machineName = source["machineName"];
	        this.host = source["host"];
	        this.user = source["user"];
	        this.isRunning = source["isRunning"];
	        this.currentCommand = source["currentCommand"];
	    }
	}
	
	
	export class SubProjectStatus {
	    projectName: string;
	    subProjectName: string;
	    isRunning: boolean;
	    currentCommand: string;
	    currentStep: string;
	    completedCommands: number;
	    completedSteps: number;
	    totalCommands: number;
	    totalSteps: number;
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
	        this.currentStep = source["currentStep"];
	        this.completedCommands = source["completedCommands"];
	        this.completedSteps = source["completedSteps"];
	        this.totalCommands = source["totalCommands"];
	        this.totalSteps = source["totalSteps"];
	        this.output = source["output"];
	    }
	}

}

export namespace keys {
	
	export class Accelerator {
	    Key: string;
	    Modifiers: string[];
	
	    static createFrom(source: any = {}) {
	        return new Accelerator(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Key = source["Key"];
	        this.Modifiers = source["Modifiers"];
	    }
	}

}

export namespace menu {
	
	export class MenuItem {
	    Label: string;
	    Role: number;
	    Accelerator?: keys.Accelerator;
	    Type: string;
	    Disabled: boolean;
	    Hidden: boolean;
	    Checked: boolean;
	    SubMenu?: Menu;
	
	    static createFrom(source: any = {}) {
	        return new MenuItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Label = source["Label"];
	        this.Role = source["Role"];
	        this.Accelerator = this.convertValues(source["Accelerator"], keys.Accelerator);
	        this.Type = source["Type"];
	        this.Disabled = source["Disabled"];
	        this.Hidden = source["Hidden"];
	        this.Checked = source["Checked"];
	        this.SubMenu = this.convertValues(source["SubMenu"], Menu);
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
	export class Menu {
	    Items: MenuItem[];
	
	    static createFrom(source: any = {}) {
	        return new Menu(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Items = this.convertValues(source["Items"], MenuItem);
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


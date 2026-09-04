export namespace app {
	
	export class AppIconPresetInfo {
	    id: string;
	    label: string;
	    preview: string;
	    isCustom: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppIconPresetInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.label = source["label"];
	        this.preview = source["preview"];
	        this.isCustom = source["isCustom"];
	    }
	}
	export class ShellCopyToOtherResult {
	    mode: string;
	    transferId?: string;
	    destPath: string;
	    sameHost: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShellCopyToOtherResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.transferId = source["transferId"];
	        this.destPath = source["destPath"];
	        this.sameHost = source["sameHost"];
	    }
	}
	export class SystemFontInfo {
	    family: string;
	    mono: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SystemFontInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.family = source["family"];
	        this.mono = source["mono"];
	    }
	}
	export class UpdateDownloadSourceInfo {
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateDownloadSourceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	    }
	}
	export class UpdateCheckResult {
	    currentVersion: string;
	    latestVersion: string;
	    hasUpdate: boolean;
	    releaseName: string;
	    releaseNotes: string;
	    releaseURL: string;
	    publishedAt: string;
	    checkedAt: string;
	    assetName?: string;
	    downloadURL?: string;
	    assetSize?: number;
	    downloadSources?: UpdateDownloadSourceInfo[];
	    downloaded?: boolean;
	    downloadPath?: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.hasUpdate = source["hasUpdate"];
	        this.releaseName = source["releaseName"];
	        this.releaseNotes = source["releaseNotes"];
	        this.releaseURL = source["releaseURL"];
	        this.publishedAt = source["publishedAt"];
	        this.checkedAt = source["checkedAt"];
	        this.assetName = source["assetName"];
	        this.downloadURL = source["downloadURL"];
	        this.assetSize = source["assetSize"];
	        this.downloadSources = this.convertValues(source["downloadSources"], UpdateDownloadSourceInfo);
	        this.downloaded = source["downloaded"];
	        this.downloadPath = source["downloadPath"];
	        this.error = source["error"];
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
	export class UpdateDownloadResult {
	    success: boolean;
	    message: string;
	    filePath?: string;
	    dirPath?: string;
	    paused?: boolean;
	    readyToInstall?: boolean;
	    installLogPath?: string;
	    autoRelaunch?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateDownloadResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.filePath = source["filePath"];
	        this.dirPath = source["dirPath"];
	        this.paused = source["paused"];
	        this.readyToInstall = source["readyToInstall"];
	        this.installLogPath = source["installLogPath"];
	        this.autoRelaunch = source["autoRelaunch"];
	    }
	}
	
	export class UpdateInstallResult {
	    success: boolean;
	    message: string;
	    logPath?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	        this.logPath = source["logPath"];
	    }
	}

}

export namespace crypto {
	
	export class Status {
	    unlocked: boolean;
	    hasMasterPassword: boolean;
	    mode: string;
	    idleLockMinutes: number;
	    keyringBackend: string;
	    unlockFailCount: number;
	    unlockLockedUntil?: string;
	    credentialAlgo: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unlocked = source["unlocked"];
	        this.hasMasterPassword = source["hasMasterPassword"];
	        this.mode = source["mode"];
	        this.idleLockMinutes = source["idleLockMinutes"];
	        this.keyringBackend = source["keyringBackend"];
	        this.unlockFailCount = source["unlockFailCount"];
	        this.unlockLockedUntil = source["unlockLockedUntil"];
	        this.credentialAlgo = source["credentialAlgo"];
	    }
	}

}

export namespace data {
	
	export class GlobalAccount {
	    id: string;
	    name: string;
	    user: string;
	    keyFile?: string;
	    encrypted_password?: string;
	    encrypted_key_passphrase?: string;
	
	    static createFrom(source: any = {}) {
	        return new GlobalAccount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.user = source["user"];
	        this.keyFile = source["keyFile"];
	        this.encrypted_password = source["encrypted_password"];
	        this.encrypted_key_passphrase = source["encrypted_key_passphrase"];
	    }
	}
	export class GlobalAccountDTO {
	    id: string;
	    name: string;
	    user: string;
	    password: string;
	    keyFile?: string;
	    keyPassphrase?: string;
	
	    static createFrom(source: any = {}) {
	        return new GlobalAccountDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.keyFile = source["keyFile"];
	        this.keyPassphrase = source["keyPassphrase"];
	    }
	}
	export class SftpFileAssociation {
	    openerType: string;
	    systemApp?: SftpSystemApp;
	
	    static createFrom(source: any = {}) {
	        return new SftpFileAssociation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.openerType = source["openerType"];
	        this.systemApp = this.convertValues(source["systemApp"], SftpSystemApp);
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
	export class SftpSystemApp {
	    path: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new SftpSystemApp(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	    }
	}
	export class ShellLogHighlightCustomKeyword {
	    keyword: string;
	    color: string;
	    enabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShellLogHighlightCustomKeyword(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.keyword = source["keyword"];
	        this.color = source["color"];
	        this.enabled = source["enabled"];
	    }
	}
	export class ShellLogHighlightColors {
	    timestamp: string;
	    threadId: string;
	    info: string;
	    debug: string;
	    warn: string;
	    error: string;
	    logger: string;
	    sql: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellLogHighlightColors(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.timestamp = source["timestamp"];
	        this.threadId = source["threadId"];
	        this.info = source["info"];
	        this.debug = source["debug"];
	        this.warn = source["warn"];
	        this.error = source["error"];
	        this.logger = source["logger"];
	        this.sql = source["sql"];
	        this.label = source["label"];
	    }
	}
	export class ProxySettings {
	    mode: string;
	    type: string;
	    host: string;
	    port: number;
	    user: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	    }
	}
	export class ThemeSettings {
	    mode: string;
	    uiAccent: string;
	    terminalPreset: string;
	    uiFontFamily: string;
	    uiFontSize: number;
	    shellFontFamily: string;
	    shellFontSize: number;
	    shellLineHeight: number;
	    shellMemorySaver: boolean;
	    shellAutoReconnect: boolean;
	    shellUseWebgl: boolean;
	    shellTabHibernate?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ThemeSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.uiAccent = source["uiAccent"];
	        this.terminalPreset = source["terminalPreset"];
	        this.uiFontFamily = source["uiFontFamily"];
	        this.uiFontSize = source["uiFontSize"];
	        this.shellFontFamily = source["shellFontFamily"];
	        this.shellFontSize = source["shellFontSize"];
	        this.shellLineHeight = source["shellLineHeight"];
	        this.shellMemorySaver = source["shellMemorySaver"];
	        this.shellAutoReconnect = source["shellAutoReconnect"];
	        this.shellUseWebgl = source["shellUseWebgl"];
	        this.shellTabHibernate = source["shellTabHibernate"];
	    }
	}
	export class MachineGroupDefaults {
	    name: string;
	    user?: string;
	    keyFile?: string;
	    proxyJump?: string;
	    startupCommand?: string;
	    sftpEncoding?: string;
	    agentForwarding?: boolean;
	    localEcho?: boolean;
	    proxyOverride?: define.MachineProxyOverride;
	    tags?: string[];
	
	    static createFrom(source: any = {}) {
	        return new MachineGroupDefaults(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.user = source["user"];
	        this.keyFile = source["keyFile"];
	        this.proxyJump = source["proxyJump"];
	        this.startupCommand = source["startupCommand"];
	        this.sftpEncoding = source["sftpEncoding"];
	        this.agentForwarding = source["agentForwarding"];
	        this.localEcho = source["localEcho"];
	        this.proxyOverride = this.convertValues(source["proxyOverride"], define.MachineProxyOverride);
	        this.tags = source["tags"];
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
	export class GlobalConfig {
	    appId: string;
	    windowsName: string;
	    configFile: string[];
	    lastOpenedFile: string;
	    lastTaskProject?: string;
	    workPaths: Record<string, string>;
	    machines?: define.Machine[];
	    machineGroups?: string[];
	    machineGroupDefaults?: MachineGroupDefaults[];
	    globalAccounts?: GlobalAccount[];
	    themeSettings: ThemeSettings;
	    proxySettings: ProxySettings;
	    shellMonitorIntervalMs: number;
	    sshHandshakeTimeoutSec: number;
	    shellTerminalScrollback: number;
	    taskOutputMaxLines: number;
	    shellCommandHistoryMax: number;
	    appIconPreset: string;
	    startupFullscreen: boolean;
	    windowOpacity: number;
	    closeToTray: boolean;
	    minimizeToTray: boolean;
	    homeMinimizedZone: string;
	    shellLogHighlight?: boolean;
	    shellLogHighlightColors: ShellLogHighlightColors;
	    shellLogHighlightDisabled: string[];
	    shellLogHighlightKeywords: ShellLogHighlightCustomKeyword[];
	    shellAsciiInput?: boolean;
	    sftpUseCompressedUpload?: boolean;
	    sftpSkipUnchanged?: boolean;
	    sftpTransferMaxConcurrent?: number;
	    shellSessionRestore?: boolean;
	    shellCursorLineHighlight?: boolean;
	    shellLineTimestamps?: boolean;
	    shellPasswordAssist?: boolean;
	    sftpDefaultOpener?: string;
	    sftpDefaultSystemApp?: SftpSystemApp;
	    sftpAutoSync?: boolean;
	    sftpFileAssociations?: Record<string, SftpFileAssociation>;
	    externalEditorCommand?: string;
	    fileAssociations?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new GlobalConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.appId = source["appId"];
	        this.windowsName = source["windowsName"];
	        this.configFile = source["configFile"];
	        this.lastOpenedFile = source["lastOpenedFile"];
	        this.lastTaskProject = source["lastTaskProject"];
	        this.workPaths = source["workPaths"];
	        this.machines = this.convertValues(source["machines"], define.Machine);
	        this.machineGroups = source["machineGroups"];
	        this.machineGroupDefaults = this.convertValues(source["machineGroupDefaults"], MachineGroupDefaults);
	        this.globalAccounts = this.convertValues(source["globalAccounts"], GlobalAccount);
	        this.themeSettings = this.convertValues(source["themeSettings"], ThemeSettings);
	        this.proxySettings = this.convertValues(source["proxySettings"], ProxySettings);
	        this.shellMonitorIntervalMs = source["shellMonitorIntervalMs"];
	        this.sshHandshakeTimeoutSec = source["sshHandshakeTimeoutSec"];
	        this.shellTerminalScrollback = source["shellTerminalScrollback"];
	        this.taskOutputMaxLines = source["taskOutputMaxLines"];
	        this.shellCommandHistoryMax = source["shellCommandHistoryMax"];
	        this.appIconPreset = source["appIconPreset"];
	        this.startupFullscreen = source["startupFullscreen"];
	        this.windowOpacity = source["windowOpacity"];
	        this.closeToTray = source["closeToTray"];
	        this.minimizeToTray = source["minimizeToTray"];
	        this.homeMinimizedZone = source["homeMinimizedZone"];
	        this.shellLogHighlight = source["shellLogHighlight"];
	        this.shellLogHighlightColors = this.convertValues(source["shellLogHighlightColors"], ShellLogHighlightColors);
	        this.shellLogHighlightDisabled = source["shellLogHighlightDisabled"];
	        this.shellLogHighlightKeywords = this.convertValues(source["shellLogHighlightKeywords"], ShellLogHighlightCustomKeyword);
	        this.shellAsciiInput = source["shellAsciiInput"];
	        this.sftpUseCompressedUpload = source["sftpUseCompressedUpload"];
	        this.sftpSkipUnchanged = source["sftpSkipUnchanged"];
	        this.sftpTransferMaxConcurrent = source["sftpTransferMaxConcurrent"];
	        this.shellSessionRestore = source["shellSessionRestore"];
	        this.shellCursorLineHighlight = source["shellCursorLineHighlight"];
	        this.shellLineTimestamps = source["shellLineTimestamps"];
	        this.shellPasswordAssist = source["shellPasswordAssist"];
	        this.sftpDefaultOpener = source["sftpDefaultOpener"];
	        this.sftpDefaultSystemApp = this.convertValues(source["sftpDefaultSystemApp"], SftpSystemApp);
	        this.sftpAutoSync = source["sftpAutoSync"];
	        this.sftpFileAssociations = this.convertValues(source["sftpFileAssociations"], SftpFileAssociation, true);
	        this.externalEditorCommand = source["externalEditorCommand"];
	        this.fileAssociations = source["fileAssociations"];
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
	export class ImportMachineTemplateResult {
	    added: number;
	    updated: number;
	    skipped: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportMachineTemplateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.skipped = source["skipped"];
	    }
	}
	export class KeyMapBinding {
	    key: string;
	    useMod: boolean;
	    useAlt?: boolean;
	    useShift?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new KeyMapBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.useMod = source["useMod"];
	        this.useAlt = source["useAlt"];
	        this.useShift = source["useShift"];
	    }
	}
	export class KeyMapEntry {
	    id: string;
	    enabled: boolean;
	    name?: string;
	    binding: KeyMapBinding;
	    action: string;
	    sendString: string;
	
	    static createFrom(source: any = {}) {
	        return new KeyMapEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.enabled = source["enabled"];
	        this.name = source["name"];
	        this.binding = this.convertValues(source["binding"], KeyMapBinding);
	        this.action = source["action"];
	        this.sendString = source["sendString"];
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
	export class KeyMapSettings {
	    entries: KeyMapEntry[];
	
	    static createFrom(source: any = {}) {
	        return new KeyMapSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entries = this.convertValues(source["entries"], KeyMapEntry);
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
	export class KnownHostRecord {
	    host: string;
	    port: number;
	    fingerprint: string;
	
	    static createFrom(source: any = {}) {
	        return new KnownHostRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.fingerprint = source["fingerprint"];
	    }
	}
	
	export class MachineImportResult {
	    imported: number;
	    skipped: number;
	    errors: string[];
	
	    static createFrom(source: any = {}) {
	        return new MachineImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.imported = source["imported"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	    }
	}
	export class OpenSSHImportResult {
	    added: number;
	    updated: number;
	    skipped: number;
	    errors?: string[];
	
	    static createFrom(source: any = {}) {
	        return new OpenSSHImportResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.skipped = source["skipped"];
	        this.errors = source["errors"];
	    }
	}
	export class PortForwardRule {
	    id: string;
	    name: string;
	    type: string;
	    localHost?: string;
	    localPort: number;
	    remoteHost?: string;
	    remotePort?: number;
	    machineName: string;
	    enabled: boolean;
	    autoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PortForwardRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.localHost = source["localHost"];
	        this.localPort = source["localPort"];
	        this.remoteHost = source["remoteHost"];
	        this.remotePort = source["remotePort"];
	        this.machineName = source["machineName"];
	        this.enabled = source["enabled"];
	        this.autoStart = source["autoStart"];
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
	
	
	
	
	export class ShellSessionRestoreTab {
	    sessionId: string;
	    configName: string;
	    kind: string;
	    tabLabel: string;
	    lastCwd?: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellSessionRestoreTab(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessionId = source["sessionId"];
	        this.configName = source["configName"];
	        this.kind = source["kind"];
	        this.tabLabel = source["tabLabel"];
	        this.lastCwd = source["lastCwd"];
	    }
	}
	export class ShellSnippet {
	    id: string;
	    name: string;
	    command: string;
	    scope?: string;
	    binding?: KeyMapBinding;
	    execute: boolean;
	    onConnect?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShellSnippet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.command = source["command"];
	        this.scope = source["scope"];
	        this.binding = this.convertValues(source["binding"], KeyMapBinding);
	        this.execute = source["execute"];
	        this.onConnect = source["onConnect"];
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
	export class ShortcutBinding {
	    key: string;
	    useMod: boolean;
	    useShift?: boolean;
	    useAlt?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ShortcutBinding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.useMod = source["useMod"];
	        this.useShift = source["useShift"];
	        this.useAlt = source["useAlt"];
	    }
	}
	export class ShortcutSettings {
	    newWindow: ShortcutBinding;
	    machineConfig: ShortcutBinding;
	    connectionManager: ShortcutBinding;
	    envVars: ShortcutBinding;
	    systemSettings: ShortcutBinding;
	    refreshConfig: ShortcutBinding;
	    find: ShortcutBinding;
	    copy: ShortcutBinding;
	    paste: ShortcutBinding;
	    clearOutput: ShortcutBinding;
	    commandPalette: ShortcutBinding;
	    quickSwitcher?: ShortcutBinding;
	    paneZoom: ShortcutBinding;
	    nextTab: ShortcutBinding;
	    prevTab: ShortcutBinding;
	    closeTab: ShortcutBinding;
	    toggleBroadcast: ShortcutBinding;
	    openSftp: ShortcutBinding;
	    openLocalShell: ShortcutBinding;
	    splitFocusLeft: ShortcutBinding;
	    splitFocusRight: ShortcutBinding;
	    splitFocusUp: ShortcutBinding;
	    splitFocusDown: ShortcutBinding;
	    snippets?: ShellSnippet[];
	
	    static createFrom(source: any = {}) {
	        return new ShortcutSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.newWindow = this.convertValues(source["newWindow"], ShortcutBinding);
	        this.machineConfig = this.convertValues(source["machineConfig"], ShortcutBinding);
	        this.connectionManager = this.convertValues(source["connectionManager"], ShortcutBinding);
	        this.envVars = this.convertValues(source["envVars"], ShortcutBinding);
	        this.systemSettings = this.convertValues(source["systemSettings"], ShortcutBinding);
	        this.refreshConfig = this.convertValues(source["refreshConfig"], ShortcutBinding);
	        this.find = this.convertValues(source["find"], ShortcutBinding);
	        this.copy = this.convertValues(source["copy"], ShortcutBinding);
	        this.paste = this.convertValues(source["paste"], ShortcutBinding);
	        this.clearOutput = this.convertValues(source["clearOutput"], ShortcutBinding);
	        this.commandPalette = this.convertValues(source["commandPalette"], ShortcutBinding);
	        this.quickSwitcher = this.convertValues(source["quickSwitcher"], ShortcutBinding);
	        this.paneZoom = this.convertValues(source["paneZoom"], ShortcutBinding);
	        this.nextTab = this.convertValues(source["nextTab"], ShortcutBinding);
	        this.prevTab = this.convertValues(source["prevTab"], ShortcutBinding);
	        this.closeTab = this.convertValues(source["closeTab"], ShortcutBinding);
	        this.toggleBroadcast = this.convertValues(source["toggleBroadcast"], ShortcutBinding);
	        this.openSftp = this.convertValues(source["openSftp"], ShortcutBinding);
	        this.openLocalShell = this.convertValues(source["openLocalShell"], ShortcutBinding);
	        this.splitFocusLeft = this.convertValues(source["splitFocusLeft"], ShortcutBinding);
	        this.splitFocusRight = this.convertValues(source["splitFocusRight"], ShortcutBinding);
	        this.splitFocusUp = this.convertValues(source["splitFocusUp"], ShortcutBinding);
	        this.splitFocusDown = this.convertValues(source["splitFocusDown"], ShortcutBinding);
	        this.snippets = this.convertValues(source["snippets"], ShellSnippet);
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
	    parallel?: boolean;
	
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
	        this.parallel = source["parallel"];
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
	export class SSHTunnel {
	    enabled: boolean;
	    name?: string;
	    type: string;
	    localHost?: string;
	    localPort: number;
	    remoteHost?: string;
	    remotePort?: number;
	    temporary?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SSHTunnel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.localHost = source["localHost"];
	        this.localPort = source["localPort"];
	        this.remoteHost = source["remoteHost"];
	        this.remotePort = source["remotePort"];
	        this.temporary = source["temporary"];
	    }
	}
	export class MachineProxyOverride {
	    mode: string;
	    type: string;
	    host: string;
	    port: number;
	    user: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new MachineProxyOverride(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	    }
	}
	export class Machine {
	    id: string;
	    encrypted_data?: string;
	    name: string;
	    group?: string;
	    key_file?: string;
	    proxyJump?: string;
	    jumpChain?: string[];
	    proxyOverride?: MachineProxyOverride;
	    legacyAlgorithms?: boolean;
	    skipEcdsaHostKey?: boolean;
	    sftpEncoding?: string;
	    sftpFileProtocol?: string;
	    sftpSudo?: boolean;
	    startupCommand?: string;
	    agentForwarding?: boolean;
	    localEcho?: boolean;
	    terminalPreset?: string;
	    pinned?: boolean;
	    tags?: string[];
	    aiPolicy?: string;
	    aiAllowlist?: string[];
	    aiAllowSudo?: boolean;
	    notes?: string;
	    icon?: string;
	    identityId?: string;
	    tunnels?: SSHTunnel[];
	    list_host?: string;
	    list_port?: number;
	    list_user?: string;
	    shellMonitorOpen?: boolean;
	    host?: string;
	    port?: number;
	    user?: string;
	
	    static createFrom(source: any = {}) {
	        return new Machine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.encrypted_data = source["encrypted_data"];
	        this.name = source["name"];
	        this.group = source["group"];
	        this.key_file = source["key_file"];
	        this.proxyJump = source["proxyJump"];
	        this.jumpChain = source["jumpChain"];
	        this.proxyOverride = this.convertValues(source["proxyOverride"], MachineProxyOverride);
	        this.legacyAlgorithms = source["legacyAlgorithms"];
	        this.skipEcdsaHostKey = source["skipEcdsaHostKey"];
	        this.sftpEncoding = source["sftpEncoding"];
	        this.sftpFileProtocol = source["sftpFileProtocol"];
	        this.sftpSudo = source["sftpSudo"];
	        this.startupCommand = source["startupCommand"];
	        this.agentForwarding = source["agentForwarding"];
	        this.localEcho = source["localEcho"];
	        this.terminalPreset = source["terminalPreset"];
	        this.pinned = source["pinned"];
	        this.tags = source["tags"];
	        this.aiPolicy = source["aiPolicy"];
	        this.aiAllowlist = source["aiAllowlist"];
	        this.aiAllowSudo = source["aiAllowSudo"];
	        this.notes = source["notes"];
	        this.icon = source["icon"];
	        this.identityId = source["identityId"];
	        this.tunnels = this.convertValues(source["tunnels"], SSHTunnel);
	        this.list_host = source["list_host"];
	        this.list_port = source["list_port"];
	        this.list_user = source["list_user"];
	        this.shellMonitorOpen = source["shellMonitorOpen"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
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
	export class ProjectSummary {
	    name: string;
	    description: string;
	    subProjectCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ProjectSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.subProjectCount = source["subProjectCount"];
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
	
	export class SSHTunnelStatus {
	    name: string;
	    type: string;
	    localHost: string;
	    localPort: number;
	    remoteHost: string;
	    remotePort: number;
	    active: boolean;
	    error?: string;
	    startedAt: number;
	    temporary?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SSHTunnelStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.localHost = source["localHost"];
	        this.localPort = source["localPort"];
	        this.remoteHost = source["remoteHost"];
	        this.remotePort = source["remotePort"];
	        this.active = source["active"];
	        this.error = source["error"];
	        this.startedAt = source["startedAt"];
	        this.temporary = source["temporary"];
	    }
	}
	export class SensitiveData {
	    host: string;
	    port: number;
	    user: string;
	    password?: string;
	    keyPassphrase?: string;
	
	    static createFrom(source: any = {}) {
	        return new SensitiveData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.keyPassphrase = source["keyPassphrase"];
	    }
	}
	export class SftpEntry {
	    name: string;
	    path: string;
	    isDir: boolean;
	    size: number;
	    mode: string;
	    modTime: number;
	    owner: string;
	    group: string;
	    type: string;
	    linkTarget?: string;
	
	    static createFrom(source: any = {}) {
	        return new SftpEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.mode = source["mode"];
	        this.modTime = source["modTime"];
	        this.owner = source["owner"];
	        this.group = source["group"];
	        this.type = source["type"];
	        this.linkTarget = source["linkTarget"];
	    }
	}
	export class SftpTransferRecord {
	    id: string;
	    machineName: string;
	    sourceMachineName?: string;
	    direction: string;
	    name: string;
	    localPath: string;
	    remotePath: string;
	    sourceRemotePath?: string;
	    isDir: boolean;
	    useCompress: boolean;
	    conflictAction: string;
	    phase?: string;
	    status: string;
	    priority: number;
	    total: number;
	    transferred: number;
	    percent: number;
	    speedBps: number;
	    error?: string;
	    startedAt: number;
	    updatedAt: number;
	    finishedAt?: number;
	
	    static createFrom(source: any = {}) {
	        return new SftpTransferRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.machineName = source["machineName"];
	        this.sourceMachineName = source["sourceMachineName"];
	        this.direction = source["direction"];
	        this.name = source["name"];
	        this.localPath = source["localPath"];
	        this.remotePath = source["remotePath"];
	        this.sourceRemotePath = source["sourceRemotePath"];
	        this.isDir = source["isDir"];
	        this.useCompress = source["useCompress"];
	        this.conflictAction = source["conflictAction"];
	        this.phase = source["phase"];
	        this.status = source["status"];
	        this.priority = source["priority"];
	        this.total = source["total"];
	        this.transferred = source["transferred"];
	        this.percent = source["percent"];
	        this.speedBps = source["speedBps"];
	        this.error = source["error"];
	        this.startedAt = source["startedAt"];
	        this.updatedAt = source["updatedAt"];
	        this.finishedAt = source["finishedAt"];
	    }
	}
	export class ShellDiskMount {
	    path: string;
	    filesystem?: string;
	    size: string;
	    used: string;
	    avail: string;
	    usePct: string;
	    usePercent: number;
	
	    static createFrom(source: any = {}) {
	        return new ShellDiskMount(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.filesystem = source["filesystem"];
	        this.size = source["size"];
	        this.used = source["used"];
	        this.avail = source["avail"];
	        this.usePct = source["usePct"];
	        this.usePercent = source["usePercent"];
	    }
	}
	export class ShellDiskList {
	    machineName: string;
	    disks: ShellDiskMount[];
	    error?: string;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new ShellDiskList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.machineName = source["machineName"];
	        this.disks = this.convertValues(source["disks"], ShellDiskMount);
	        this.error = source["error"];
	        this.updatedAt = source["updatedAt"];
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
	
	export class ShellHistoryRecord {
	    machineId: string;
	    machineName: string;
	    host: string;
	    port: number;
	    user: string;
	    lastConnectedAt: number;
	    connectCount: number;
	
	    static createFrom(source: any = {}) {
	        return new ShellHistoryRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.machineId = source["machineId"];
	        this.machineName = source["machineName"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.lastConnectedAt = source["lastConnectedAt"];
	        this.connectCount = source["connectCount"];
	    }
	}
	export class ShellListenPort {
	    proto: string;
	    address: string;
	    port: string;
	    pid: string;
	    process: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellListenPort(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proto = source["proto"];
	        this.address = source["address"];
	        this.port = source["port"];
	        this.pid = source["pid"];
	        this.process = source["process"];
	    }
	}
	export class ShellListenPortList {
	    machineName: string;
	    host: string;
	    ports: ShellListenPort[];
	    error?: string;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new ShellListenPortList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.machineName = source["machineName"];
	        this.host = source["host"];
	        this.ports = this.convertValues(source["ports"], ShellListenPort);
	        this.error = source["error"];
	        this.updatedAt = source["updatedAt"];
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
	export class ShellProcessStat {
	    pid: string;
	    user: string;
	    cpu: number;
	    mem: number;
	    memRss: string;
	    command: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellProcessStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pid = source["pid"];
	        this.user = source["user"];
	        this.cpu = source["cpu"];
	        this.mem = source["mem"];
	        this.memRss = source["memRss"];
	        this.command = source["command"];
	    }
	}
	export class ShellMonitorSnapshot {
	    machineName: string;
	    host: string;
	    uptimeSec: number;
	    uptimeText: string;
	    cpuPercent: number;
	    memPercent: number;
	    memUsed: string;
	    memTotal: string;
	    swapPercent: number;
	    swapUsed: string;
	    swapTotal: string;
	    topMem: ShellProcessStat[];
	    netIface: string;
	    netIfaces: string[];
	    netRxRate: number;
	    netTxRate: number;
	    netRxText: string;
	    netTxText: string;
	    error?: string;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new ShellMonitorSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.machineName = source["machineName"];
	        this.host = source["host"];
	        this.uptimeSec = source["uptimeSec"];
	        this.uptimeText = source["uptimeText"];
	        this.cpuPercent = source["cpuPercent"];
	        this.memPercent = source["memPercent"];
	        this.memUsed = source["memUsed"];
	        this.memTotal = source["memTotal"];
	        this.swapPercent = source["swapPercent"];
	        this.swapUsed = source["swapUsed"];
	        this.swapTotal = source["swapTotal"];
	        this.topMem = this.convertValues(source["topMem"], ShellProcessStat);
	        this.netIface = source["netIface"];
	        this.netIfaces = source["netIfaces"];
	        this.netRxRate = source["netRxRate"];
	        this.netTxRate = source["netTxRate"];
	        this.netRxText = source["netRxText"];
	        this.netTxText = source["netTxText"];
	        this.error = source["error"];
	        this.updatedAt = source["updatedAt"];
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
	export class ShellProcessList {
	    machineName: string;
	    host: string;
	    processes: ShellProcessStat[];
	    error?: string;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new ShellProcessList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.machineName = source["machineName"];
	        this.host = source["host"];
	        this.processes = this.convertValues(source["processes"], ShellProcessStat);
	        this.error = source["error"];
	        this.updatedAt = source["updatedAt"];
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
	
	export class ShellStatus {
	    connected: boolean;
	    connecting: boolean;
	    machineName: string;
	    configName: string;
	    tabLabel: string;
	    host: string;
	    user: string;
	    isRunning: boolean;
	    currentCommand: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connected = source["connected"];
	        this.connecting = source["connecting"];
	        this.machineName = source["machineName"];
	        this.configName = source["configName"];
	        this.tabLabel = source["tabLabel"];
	        this.host = source["host"];
	        this.user = source["user"];
	        this.isRunning = source["isRunning"];
	        this.currentCommand = source["currentCommand"];
	        this.kind = source["kind"];
	    }
	}
	export class ShellSystemInfo {
	    machineName: string;
	    host: string;
	    hostname: string;
	    os: string;
	    kernel: string;
	    arch: string;
	    cpuModel: string;
	    diskSummary: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellSystemInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.machineName = source["machineName"];
	        this.host = source["host"];
	        this.hostname = source["hostname"];
	        this.os = source["os"];
	        this.kernel = source["kernel"];
	        this.arch = source["arch"];
	        this.cpuModel = source["cpuModel"];
	        this.diskSummary = source["diskSummary"];
	        this.error = source["error"];
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

export namespace machine {
	
	export class DryRunLine {
	    commandName: string;
	    commandType: string;
	    machine?: string;
	    workdir?: string;
	    stepIndex: number;
	    stepCommand: string;
	    whenExpr?: string;
	    whenResult?: string;
	    skipped: boolean;
	    parallel?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DryRunLine(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.commandName = source["commandName"];
	        this.commandType = source["commandType"];
	        this.machine = source["machine"];
	        this.workdir = source["workdir"];
	        this.stepIndex = source["stepIndex"];
	        this.stepCommand = source["stepCommand"];
	        this.whenExpr = source["whenExpr"];
	        this.whenResult = source["whenResult"];
	        this.skipped = source["skipped"];
	        this.parallel = source["parallel"];
	    }
	}
	export class LocalShellOption {
	    id: string;
	    name: string;
	    command: string;
	    isDefault: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LocalShellOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.command = source["command"];
	        this.isDefault = source["isDefault"];
	    }
	}
	export class SftpUploadConflict {
	    remotePath: string;
	    localSize: number;
	    remoteSize: number;
	    remoteMtime: number;
	    isDir: boolean;
	    localIsDir: boolean;
	    existingType: string;
	
	    static createFrom(source: any = {}) {
	        return new SftpUploadConflict(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.remotePath = source["remotePath"];
	        this.localSize = source["localSize"];
	        this.remoteSize = source["remoteSize"];
	        this.remoteMtime = source["remoteMtime"];
	        this.isDir = source["isDir"];
	        this.localIsDir = source["localIsDir"];
	        this.existingType = source["existingType"];
	    }
	}

}

export namespace mcp {
	
	export class ApprovalItem {
	    id: string;
	    tool: string;
	    toolDesc?: string;
	    server: string;
	    preview: string;
	    summary: string;
	    paramsJson?: string;
	    source: string;
	    reason?: string;
	    outboundHosts?: string[];
	    createdAt: string;
	    expiresAt: string;
	    remainingSecs: number;
	    isDanger: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ApprovalItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.tool = source["tool"];
	        this.toolDesc = source["toolDesc"];
	        this.server = source["server"];
	        this.preview = source["preview"];
	        this.summary = source["summary"];
	        this.paramsJson = source["paramsJson"];
	        this.source = source["source"];
	        this.reason = source["reason"];
	        this.outboundHosts = source["outboundHosts"];
	        this.createdAt = source["createdAt"];
	        this.expiresAt = source["expiresAt"];
	        this.remainingSecs = source["remainingSecs"];
	        this.isDanger = source["isDanger"];
	    }
	}
	export class AuditEntry {
	    id: string;
	    time: string;
	    source: string;
	    caller: string;
	    tool: string;
	    module: string;
	    server: string;
	    params: string;
	    result: string;
	    decision: string;
	    reason: string;
	    approver?: string;
	    durationMs: number;
	    sensitiveIds?: string[];
	
	    static createFrom(source: any = {}) {
	        return new AuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = source["time"];
	        this.source = source["source"];
	        this.caller = source["caller"];
	        this.tool = source["tool"];
	        this.module = source["module"];
	        this.server = source["server"];
	        this.params = source["params"];
	        this.result = source["result"];
	        this.decision = source["decision"];
	        this.reason = source["reason"];
	        this.approver = source["approver"];
	        this.durationMs = source["durationMs"];
	        this.sensitiveIds = source["sensitiveIds"];
	    }
	}
	export class AuditFilter {
	    tool: string;
	    module: string;
	    server: string;
	    source: string;
	    caller: string;
	    decision: string;
	    keyword: string;
	    startTime: string;
	    endTime: string;
	    limit: number;
	
	    static createFrom(source: any = {}) {
	        return new AuditFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool = source["tool"];
	        this.module = source["module"];
	        this.server = source["server"];
	        this.source = source["source"];
	        this.caller = source["caller"];
	        this.decision = source["decision"];
	        this.keyword = source["keyword"];
	        this.startTime = source["startTime"];
	        this.endTime = source["endTime"];
	        this.limit = source["limit"];
	    }
	}
	export class AuditMeta {
	    tools: string[];
	    modules: string[];
	    servers: string[];
	    sources: string[];
	
	    static createFrom(source: any = {}) {
	        return new AuditMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tools = source["tools"];
	        this.modules = source["modules"];
	        this.servers = source["servers"];
	        this.sources = source["sources"];
	    }
	}
	export class AuditStats {
	    total: number;
	    today: number;
	    auto: number;
	    approved: number;
	    denied: number;
	    blocked: number;
	    cancelled: number;
	    pending: number;
	
	    static createFrom(source: any = {}) {
	        return new AuditStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.today = source["today"];
	        this.auto = source["auto"];
	        this.approved = source["approved"];
	        this.denied = source["denied"];
	        this.blocked = source["blocked"];
	        this.cancelled = source["cancelled"];
	        this.pending = source["pending"];
	    }
	}
	export class ClientLink {
	    id: string;
	    name: string;
	    desc: string;
	    config: string;
	    linked: boolean;
	    configPath: string;
	    installed: boolean;
	    guidanceOk: boolean;
	    guidancePath: string;
	
	    static createFrom(source: any = {}) {
	        return new ClientLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.desc = source["desc"];
	        this.config = source["config"];
	        this.linked = source["linked"];
	        this.configPath = source["configPath"];
	        this.installed = source["installed"];
	        this.guidanceOk = source["guidanceOk"];
	        this.guidancePath = source["guidancePath"];
	    }
	}
	export class ClientSnippet {
	    stdioJson: string;
	    httpJson: string;
	    toml: string;
	    httpUrl: string;
	    httpAuth: string;
	
	    static createFrom(source: any = {}) {
	        return new ClientSnippet(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.stdioJson = source["stdioJson"];
	        this.httpJson = source["httpJson"];
	        this.toml = source["toml"];
	        this.httpUrl = source["httpUrl"];
	        this.httpAuth = source["httpAuth"];
	    }
	}
	export class DangerRule {
	    pattern: string;
	    label: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new DangerRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pattern = source["pattern"];
	        this.label = source["label"];
	        this.kind = source["kind"];
	    }
	}
	export class FalsePositiveSample {
	    id: string;
	    rule: string;
	    kind: string;
	    hash?: string;
	    server?: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new FalsePositiveSample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.rule = source["rule"];
	        this.kind = source["kind"];
	        this.hash = source["hash"];
	        this.server = source["server"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class GuidanceStatus {
	    ok: boolean;
	    stale: boolean;
	    path: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new GuidanceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.stale = source["stale"];
	        this.path = source["path"];
	        this.version = source["version"];
	    }
	}
	export class InstallOpts {
	    tokenName: string;
	    servers: string[];
	    cidrs: string[];
	
	    static createFrom(source: any = {}) {
	        return new InstallOpts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tokenName = source["tokenName"];
	        this.servers = source["servers"];
	        this.cidrs = source["cidrs"];
	    }
	}
	export class IssueOpts {
	    name: string;
	    client: string;
	    servers: string[];
	    cidrs: string[];
	
	    static createFrom(source: any = {}) {
	        return new IssueOpts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.client = source["client"];
	        this.servers = source["servers"];
	        this.cidrs = source["cidrs"];
	    }
	}
	export class IssuedToken {
	    id: string;
	    name: string;
	    client: string;
	    servers: string[];
	    cidrs: string[];
	    createdAt: string;
	    lastUsedAt?: string;
	    token: string;
	
	    static createFrom(source: any = {}) {
	        return new IssuedToken(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.client = source["client"];
	        this.servers = source["servers"];
	        this.cidrs = source["cidrs"];
	        this.createdAt = source["createdAt"];
	        this.lastUsedAt = source["lastUsedAt"];
	        this.token = source["token"];
	    }
	}
	export class PromoteSensitiveOpts {
	    id: string;
	    server: string;
	    kind: string;
	    label: string;
	    field: string;
	
	    static createFrom(source: any = {}) {
	        return new PromoteSensitiveOpts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.server = source["server"];
	        this.kind = source["kind"];
	        this.label = source["label"];
	        this.field = source["field"];
	    }
	}
	export class RedactHit {
	    rule: string;
	    kind: string;
	    start: number;
	    end: number;
	    match: string;
	    secret: string;
	    snippet: string;
	
	    static createFrom(source: any = {}) {
	        return new RedactHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rule = source["rule"];
	        this.kind = source["kind"];
	        this.start = source["start"];
	        this.end = source["end"];
	        this.match = source["match"];
	        this.secret = source["secret"];
	        this.snippet = source["snippet"];
	    }
	}
	export class Settings {
	    enabled: boolean;
	    autoStart: boolean;
	    httpPort: number;
	    bindLan: boolean;
	    defaultPolicy: string;
	    aiMode: string;
	    armedUntil: string;
	    emergencyStop: boolean;
	    auditRetentionDays: number;
	    outboundAllowlistDisabled: boolean;
	    outboundAllowlistEnabled: boolean;
	    outboundHosts: string[];
	    redactionTTLDays: number;
	    customDangerPatterns: string[];
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.autoStart = source["autoStart"];
	        this.httpPort = source["httpPort"];
	        this.bindLan = source["bindLan"];
	        this.defaultPolicy = source["defaultPolicy"];
	        this.aiMode = source["aiMode"];
	        this.armedUntil = source["armedUntil"];
	        this.emergencyStop = source["emergencyStop"];
	        this.auditRetentionDays = source["auditRetentionDays"];
	        this.outboundAllowlistDisabled = source["outboundAllowlistDisabled"];
	        this.outboundAllowlistEnabled = source["outboundAllowlistEnabled"];
	        this.outboundHosts = source["outboundHosts"];
	        this.redactionTTLDays = source["redactionTTLDays"];
	        this.customDangerPatterns = source["customDangerPatterns"];
	    }
	}
	export class Status {
	    enabled: boolean;
	    autoStart: boolean;
	    online: boolean;
	    stdioOnline: boolean;
	    observerOnline: boolean;
	    httpUrl: string;
	    httpPort: number;
	    bindLan: boolean;
	    lanUrl: string;
	    localAddr: string;
	    stdioPath: string;
	    cursorLinked: boolean;
	    tokenCount: number;
	    toolCount: number;
	    serverCount: number;
	    pendingCount: number;
	    defaultPolicy: string;
	    startedAt: string;
	    clients: ClientLink[];
	    instructions: string[];
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.autoStart = source["autoStart"];
	        this.online = source["online"];
	        this.stdioOnline = source["stdioOnline"];
	        this.observerOnline = source["observerOnline"];
	        this.httpUrl = source["httpUrl"];
	        this.httpPort = source["httpPort"];
	        this.bindLan = source["bindLan"];
	        this.lanUrl = source["lanUrl"];
	        this.localAddr = source["localAddr"];
	        this.stdioPath = source["stdioPath"];
	        this.cursorLinked = source["cursorLinked"];
	        this.tokenCount = source["tokenCount"];
	        this.toolCount = source["toolCount"];
	        this.serverCount = source["serverCount"];
	        this.pendingCount = source["pendingCount"];
	        this.defaultPolicy = source["defaultPolicy"];
	        this.startedAt = source["startedAt"];
	        this.clients = this.convertValues(source["clients"], ClientLink);
	        this.instructions = source["instructions"];
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
	export class Token {
	    id: string;
	    name: string;
	    client: string;
	    servers: string[];
	    cidrs: string[];
	    createdAt: string;
	    lastUsedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new Token(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.client = source["client"];
	        this.servers = source["servers"];
	        this.cidrs = source["cidrs"];
	        this.createdAt = source["createdAt"];
	        this.lastUsedAt = source["lastUsedAt"];
	    }
	}
	export class UpdateTokenOpts {
	    id: string;
	    name: string;
	    servers: string[];
	    cidrs: string[];
	
	    static createFrom(source: any = {}) {
	        return new UpdateTokenOpts(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.servers = source["servers"];
	        this.cidrs = source["cidrs"];
	    }
	}
	export class UserRedactRule {
	    name: string;
	    pattern: string;
	    kind: string;
	
	    static createFrom(source: any = {}) {
	        return new UserRedactRule(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.pattern = source["pattern"];
	        this.kind = source["kind"];
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


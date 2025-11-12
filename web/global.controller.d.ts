//#region service

interface ServiceContext {
    "Native Service Context"; /* it is not allowed to create it by yourself */
    getHeader(): { [name: string]: string; };
    getURL(): { path: string; params: { [name: string]: string[]; }; };
    getBody(): Buffer;
    getMethod(): string;
    getForm(): { [name: string]: string[]; };
    getPathVariables(): { [name: string]: string; };
    getFile(name: string): { name: string; size: number; data: Buffer; };
    getCerts(): any[];
    getCookie(name: string): { value: string; };
    upgradeToWebSocket(): WebSocket;
    getReader(): { readByte(): number; read(count: number): Buffer; };
    getPusher(): { push(target: string, options: any): void; };
    write(data: GenericByteArray): number;
    flush(): void;
    resetTimeout(timeout: number): void;
}

//#endregion

//#region builtin

declare class ServiceResponse {
    constructor(status: number, header?: { [name: string]: string | number; }, data?: any);
    setStatus(status: number): void;
    setHeader(name: string, value: string): void;
    setData(data: any): void;
    setCookie(name: string, value: string): void;
}

//#endregion

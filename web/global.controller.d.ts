//#region service

interface ServiceContext {
    "Native Service Context"; /* do not instantiate directly */
    /**
     * get request headers
     * 
     * @return headers object
     */
    getHeader(): { [name: string]: string; };
    /**
     * get request URL path and parameters
     * 
     * @return object with path and params
     */
    getURL(): { path: string; params: { [name: string]: string[]; }; };
    /**
     * get request body
     * 
     * @return body buffer
     */
    getBody(): Buffer;
    /**
     * get request method
     * 
     * @return method string, e.g. "GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS" etc.
     */
    getMethod(): string;
    /**
     * get form data
     * 
     * @return form object
     */
    getForm(): { [name: string]: string[]; };
    /**
     * get path variables
     * 
     * @return path variables object
     */
    getPathVariables(): { [name: string]: string; };
    /**
     * get uploaded file by name
     * 
     * @param name name of the uploaded file
     * @return object with name, size and data buffer of the file
     */
    getFile(name: string): { name: string; size: number; data: Buffer; };
    /**
     * get client certificates
     * 
     * @return array of certificates
     */
    getCerts(): any[];
    /**
     * get cookie by name
     * 
     * @param name name of the cookie
     * @return object with value of the cookie
     */
    getCookie(name: string): { value: string; };
    /**
     * upgrade the HTTP connection to WebSocket
     * 
     * @return WebSocket object
     */
    upgradeToWebSocket(): WebSocket;
    /**
     * get reader for reading request body in streaming mode
     * 
     * @return reader object with readByte and read methods
     */
    getReader(): { readByte(): number; read(count: number): Buffer; };
    /**
     * get pusher for writing response body in streaming mode
     * 
     * @return pusher object with push method
     */
    getPusher(): { push(target: string, options: any): void; };
    /**
     * write data to the response body in streaming mode
     * 
     * @param data data
     * @return number of bytes written
     */
    write(data: GenericByteArray): number;
    /**
     * flush the response buffer
     * 
     * @return void
     */
    flush(): void;
    /**
     * set/reset the timeout of the service context
     * 
     * @param timeout timeout in milliseconds
     * @return void
     */
    resetTimeout(timeout: number): void;
}

//#endregion

//#region builtin

declare class ServiceResponse {
    /**
     * create service response
     * 
     * @param status status code
     * @param header header
     * @param data data
     * @return service response
     */
    constructor(status: number, header?: { [name: string]: string | number; }, data?: any);
    /**
     * set status code
     * 
     * @param status status code
     * @return void
     */
    setStatus(status: number): void;
    /**
     * set response header
     * 
     * @param name header name
     * @param value header value
     * @return void
     */
    setHeader(name: string, value: string): void;
    /**
     * set response data
     * 
     * @param data data
     * @return void
     */
    setData(data: any): void;
    /**
     * set cookie
     * 
     * @param name cookie name
     * @param value cookie value
     * @return void
     */
    setCookie(name: string, value: string): void;
}

//#endregion

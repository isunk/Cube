type GenericByteArray = string | Uint8Array | Array<number> | Buffer

//#region builtin

declare interface Buffer extends Array<number> {
    /**
     * parse buffer to string
     * 
     * @param encoding encoding
     * @return string
     */
    toString(encoding?: "utf8" | "hex" | "base64" | "base64url"): string;
    /**
     * parse buffer to json object
     * 
     * @return json object
     */
    toJson(): any;
}
declare interface BufferConstructor {
    /**
     * parse input to buffer
     * 
     * @param input input data
     * @param encoding encoding if input is string
     * @return buffer object
     */
    from(input: GenericByteArray, encoding?: "utf8" | "hex" | "base64" | "base64url"): Buffer;
}
declare var Buffer: BufferConstructor;

interface Console {
    log(...data: any[]): void;
    debug(...data: any[]): void;
    info(...data: any[]): void;
    warn(...data: any[]): void;
    error(...data: any[]): void;
}
declare var console: Console;

interface Date {
    /**
     * parse date to string
     * 
     * @param layout date layout, e.g. "yyyy-MM-dd HH:mm:ss.SSS"
     * @return date string
     */
    toString(layout?: string): string
}
interface DateConstructor {
    /**
     * parse string to date
     * 
     * @param value date string
     * @param layout date layout, e.g. "yyyy-MM-dd HH:mm:ss.SSS"
     * @return date object
     */
    toDate(value: string, layout: string): Date
}

type DatabaseTransaction = {
    /**
     * query data
     * 
     * @param stmt statement
     * @param params parameters
     * @return query result rows
     */
    query(stmt: string, ...params: any[]): any[];
    /**
     * execute statement
     * 
     * @param stmt statement
     * @param params parameters
     * @return number of affected rows
     */
    exec(stmt: string, ...params: any[]): number;
    /**
     * commit this transaction
     * 
     * @return void
     */
    commit(): void;
    /**
     * rollback this transaction
     * 
     * @return void
     */
    rollback(): void;
}
declare class Database {
    /**
     * create a database client
     * 
     * @param type type, e.g. "sqlite3", "mysql"
     * @param connection connection string, e.g. "mydb.db" for sqlite3, "username:password@tcp(127.0.0.1:3307)/dbname" for mysql
     * @return database client
     */
    constructor(type: "sqlite3" | "mysql", connection: string);
    /**
     * begin a transaction
     *
     * @param func function during this transaction
     * @param isolation transaction isolation level: 0 = Default, 1 = Read Uncommitted, 2 = Read Committed, 3 = Write Committed, 4 = Repeatable Read, 5 = Snapshot, 6 = Serializable, 7 = Linearizable
     * @return void
     */
    transaction(func: (tx: DatabaseTransaction) => void, isolation: number = 0): void;
    /**
     * query data
     * 
     * @param stmt statement
     * @param params parameters
     * @return query result rows
     */
    query(stmt: string, ...params: any[]): any[];
    /**
     * execute statement
     * 
     * @param stmt statement
     * @param params parameters
     * @return number of affected rows
     */
    exec(stmt: string, ...params: any[]): number;
}

declare class Decimal {
    constructor(value: string);
    add(value: Decimal): Decimal;
    sub(value: Decimal): Decimal;
    mul(value: Decimal): Decimal;
    div(value: Decimal): Decimal;
    pow(value: Decimal): Decimal;
    mod(value: Decimal): Decimal;
    compare(value: Decimal): -1 | 0 | 1;
    abs(): Decimal;
    string(): string;
    stringFixed(places: number): string;
}

interface IntervalId { "Native Interval Id"; }
/**
 * set an interval timer
 * 
 * @param handler handler function
 * @param timeout timeout in milliseconds
 * @param arguments arguments to pass to handler
 * @return interval id
 */
declare function setInterval(handler: Function, timeout?: number, ...arguments: any[]): IntervalId;
/**
 * clear an interval timer
 * 
 * @param id interval id
 * @return void
 */
declare function clearInterval(id: IntervalId): void;
interface TimeoutId { "Native Timeout Id"; }
/**
 * set a timeout timer
 * 
 * @param handler handler function
 * @param timeout timeout in milliseconds
 * @param arguments arguments to pass to handler
 * @return timeout id
 */
declare function setTimeout(handler: Function, timeout?: number, ...arguments: any[]): TimeoutId;
/**
 * clear a timeout timer
 * 
 * @param id timeout id
 * @return void
 */
declare function clearTimeout(id: TimeoutId): void;

/**
 * fetch url with options
 * 
 * @param url url
 * @param options options with method, headers and body
 * @return promise of response with status, headers and methods to get buffer, json and text
 */
declare function fetch(url: string, options?: { method?: "GET" | "POST" | "PUT" | "DELETE"; headers?: { [name: string]: string }; body?: string; }): Promise<{ status: number; headers: { [name: string]: string }; buffer(): Buffer; json(): any; text(): string; }>;

interface WebSocket {
    /**
     * read a message from the WebSocket
     * 
     * @return message with type and data
     */
    read(): { messageType: number; data: Buffer; };
    /**
     * send a message to the WebSocket
     * 
     * @param data data to send
     * @return void
     */
    send(data: GenericByteArray): void;
    /**
     * close the WebSocket connection
     * 
     * @return void
     */
    close(): void;
}
declare var WebSocket: {
    prototype: WebSocket;
    /**
     * create a WebSocket connection
     * 
     * @param url url
     * @return WebSocket object
     */
    new(url: string): WebSocket;
}

//#endregion

//#region native module

type BlockingQueue = {
    /**
     * put input to the queue, block until timeout
     * 
     * @param input input
     * @param timeout timeout in milliseconds
     * @return void
     */
    put(input: any, timeout: number): void;
    /**
     * poll an item from the queue, block until timeout
     * 
     * @param timeout timeout in milliseconds
     * @return item or null if timeout
     */
    poll(timeout: number): any;
    /**
     * drain multiple items from the queue, block until timeout
     * 
     * @param size size
     * @param timeout timeout in milliseconds
     * @return array of items
     */
    drain(size: number, timeout: number): any[];
}
declare function $native(name: "bqueue"): (size: number) => BlockingQueue;

declare function $native(name: "cache"): {
    /**
     * set key-value with timeout
     * 
     * @param key key
     * @param value value
     * @param timeout timeout in milliseconds
     * @return void
     */
    set(key: any, value: any, timeout: number): void;
    /**
     * get value by key
     * 
     * @param key key
     * @return value
     */
    get(key: any): any;
    /**
     * check whether the key exists
     * 
     * @param key key
     * @return whether the key exists
     */
    has(key: any): boolean;
    /**
     * expire the key with timeout
     * 
     * @param key key
     * @param timeout timeout in milliseconds
     * @return void
     */
    expire(key: any, timeout: number): void;
}

type HashAlgorithm = "md5" | "sha1" | "sha256" | "sha512"
declare function $native(name: "crypto"): {
    /**
     * create cipher for encryption and decryption
     * 
     * @param algorithm algorithm, e.g. "aes-ecb"
     * @param key key
     * @param options options
     * @return cipher object
     */
    createCipher(algorithm: "aes-ecb", key: GenericByteArray, options: { padding: "none" | "pkcs5" | "pkcs7"; }): {
        /**
         * encrypt input data
         * 
         * @param input input data
         * @return encrypted data
         */
        encrypt(input: GenericByteArray): Buffer;
        /**
         * decrypt input data
         * 
         * @param input input data
         * @return decrypted data
         */
        decrypt(input: GenericByteArray): Buffer;
    };
    /**
     * create hash object
     * 
     * @param algorithm algorithm
     * @return hash object
     */
    createHash(algorithm: HashAlgorithm): {
        /**
         * sum input data
         * 
         * @param input input data
         * @return hash value
         */
        sum(input: GenericByteArray): Buffer;
    };
    /**
     * create HMAC object
     * 
     * @param algorithm algorithm
     * @return HMAC object
     */
    createHmac(algorithm: HashAlgorithm): {
        /**
         * sum input data with key
         * 
         * @param input input data
         * @param key key
         * @return HMAC value
         */
        sum(input: GenericByteArray, key: GenericByteArray): Buffer;
    };
    /**
     * create RSA object
     * 
     * @return RSA object
     */
    createRsa(): {
        /**
         * generate RSA key pair
         * 
         * @return key pair with privateKey and publicKey
         */
        generateKey(): { privateKey: Buffer; publicKey: Buffer; };
        /**
         * encrypt input data with public key
         * 
         * @param input input data
         * @param publicKey public key
         * @param padding padding scheme
         * @return encrypted data
         */
        encrypt(input: GenericByteArray, publicKey: GenericByteArray, padding: "pkcs1" | "oaep" = "pkcs1"): Buffer;
        /**
         * decrypt input data with private key
         * 
         * @param input input data
         * @param privateKey private key
         * @param padding padding scheme
         * @return decrypted data
         */
        decrypt(input: GenericByteArray, privateKey: GenericByteArray, padding: "pkcs1" | "oaep" = "pkcs1"): Buffer;
        /**
         * sign input data with private key
         * 
         * @param input input data
         * @param key key
         * @param algorithm algorithm
         * @param padding padding scheme
         * @return signature
         */
        sign(input: GenericByteArray, key: GenericByteArray, algorithm: HashAlgorithm, padding: "pkcs1" | "pss" = "pkcs1"): Buffer;
        /**
         * verify signature with public key
         * 
         * @param input input data
         * @param sign signature
         * @param key key
         * @param algorithm algorithm
         * @param padding padding scheme
         * @return whether the signature is valid
         */
        verify(input: GenericByteArray, sign: GenericByteArray, key: GenericByteArray, algorithm: HashAlgorithm, padding: "pkcs1" | "pss" = "pkcs1"): boolean;
    };
}

declare function $native(name: "db"): Database;

declare function $native(name: "email"): (host: string, port: number, username: string, password: string) => {
    /**
     * send an email
     * 
     * @param receivers receivers
     * @param subject subject
     * @param content content
     * @param attachments array of attachments with Name, ContentType and Base64 fields
     * @return void
     */
    send(receivers: string[], subject: string, content: string, attachments: { Name: string; ContentType: string; Base64: string; }[]): void;
}

declare function $native(name: "event"): {
    /**
     * emit an event with topic and data
     * 
     * @param topic topic
     * @param data data
     * @return void
     */
    emit(topic: string, data: any): void;
    /**
     * create a subscriber for given topics
     * 
     * @param topics topics
     * @return subscriber object with next method
     */
    createSubscriber(...topics: string[]): {
        /**
         * next event data
         * 
         * @return event data
         */
        next(): any;
    };
    /**
     * listen on a topic with a callback function
     * 
     * @param topic topic
     * @param func function to handle event data
     * @return object with cancel method
     */
    on(topic: string, func: (data: any) => void): {
        /**
         * cancel this listener
         * 
         * @return void
         */
        cancel(): void;
    };
}

declare function $native(name: "file"): {
    /**
     * read file content
     * 
     * @param name name of the file
     * @return file content
     */
    read(name: string): Buffer;
    /**
     * read a range of file content
     * 
     * @param name name of the file
     * @param offset offset
     * @param length length
     * @return file content
     */
    readRange(name: string, offset: number, length: number): Buffer;
    /**
     * write content to file
     * 
     * @param name name of the file
     * @param content content to write
     * @return void
     */
    write(name: string, content: GenericByteArray): void;
    /**
     * write a range of content to file
     * 
     * @param name name of the file
     * @param offset offset
     * @param content content to write
     * @return void
     */
    writeRange(name: string, offset: number, content: GenericByteArray): void;
    /**
     * stat file or directory
     * 
     * @param name name of the file or directory
     * @return stat object
     */
    stat(name: string): {
        /**
         * name of the file or directory
         * 
         * @return name
         */
        name(): string;
        /**
         * size of the file or directory
         * 
         * @return size
         */
        size(): number;
        /**
         * whether it is a directory
         * 
         * @return whether it is a directory
         */
        isDir(): boolean;
        /**
         * mode of the file or directory
         * 
         * @return mode
         */
        mode(): string;
        /**
         * modification time of the file or directory
         * 
         * @return modification time
         */
        modTime(): string;
    };
    /**
     * list files in a directory
     * 
     * @param name name of the directory
     * @return array of file names
     */
    list(name: string): string[];
    /**
     * remove a file or directory
     * 
     * @param name name of the file or directory
     * @return void
     */
    remove(name: string): void;
}

type HttpOptions = Partial<{
    /**
     * ca cert for https request
     */
    caCert: string;
    /**
     * proxy url for http request
     */
    proxy: string;
    /**
     * whether to skip insecure verify for https request
     */
    isSkipInsecureVerify: boolean;
    /**
     * whether to use http3 for http request
     */
    isHttp3: boolean;
    /**
     * whether to not follow redirect for http request
     */
    isNotFollowRedirect: boolean;
}> | {
    /**
     * client cert for https request
     */
    cert: string;
    /**
     * client key for https request
     */
    key: string;
}
type FormData = {
    "Native Form Data"
}
declare function $native(name: "http"): (options?: HttpOptions) => {
    /**
     * send http request
     * 
     * @param method method, e.g. "GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS" etc.
     * @param url url
     * @param header headers
     * @param body body
     * @return response with status, header and data
     */
    request(method: Uppercase<string>, url: string, header?: { [name: string]: string; }, body?: GenericByteArray | FormData): { status: number; header: { [name: string]: string; }; data: Buffer; };
    /**
     * parse data to form data
     * 
     * @param data data object with string or file content
     * @return form data
     */
    toFormData(data: { [name: string]: string | { filename: string; data: GenericByteArray; }; }): FormData;
}

type Image = {
    /**
     * width of the image
     * 
     * @return width
     */
    width(): number;
    /**
     * height of the image
     * 
     * @return height
     */
    height(): number;
    /**
     * get pixel RGBA at (x, y)
     * 
     * @param x x
     * @param y y
     * @return RGBA array
     */
    get(x: number, y: number): [number, number, number, number];
    /**
     * set pixel RGBA at (x, y)
     * 
     * @param x x
     * @param y y
     * @param rgba rgba array
     * @return void
     */
    set(x: number, y: number, rgba: [number, number, number, number]): void;
    /**
     * set rotation for next drawings
     * 
     * @param degrees rotation degrees
     * @return void
     */
    setDrawRotate(degrees: number): void;
    /**
     * set font face for next drawings
     * 
     * @param fontSize font size
     * @param ttf true type font data
     * @return void
     */
    setDrawFontFace(fontSize?: number, ttf?: GenericByteArray): void;
    /**
     * set color for next drawings
     * 
     * @param color color string like "#RRGGBB" or "#RRGGBBAA", or RGBA array
     * @return void
     */
    setDrawColor(color: string | [red: number, green: number, blue: number, alpha?: number]): void;
    /**
     * get string width and height with current font face
     * 
     * @param s string
     * @return width and height
     */
    getStringWidthAndHeight(s: string): { width: number; height: number; };
    /**
     * draw image at position (x, y)
     * 
     * @param image image to draw
     * @param x x
     * @param y y
     * @return void
     */
    drawImage(image: Image, x: number, y: number): void;
    /**
     * draw string at position (x, y) with alignment and wrapping
     * 
     * @param s string
     * @param x x
     * @param y y
     * @param ax ax alignment x: 0 = left, 0.5 = center, 1 = right
     * @param ay ay alignment y: 0 = top, 0.5 = middle, 1 = bottom
     * @param width width for wrapping
     * @param lineSpacing line spacing
     * @return void
     */
    drawString(s: string, x: number, y: number, ax?: number, ay?: number, width?: number, lineSpacing?: number): void;
    /**
     * crop image
     * 
     * @param sx sx
     * @param sy sy
     * @param ex ex
     * @param ey ey
     * @return cropped image
     */
    crop(sx: number, sy: number, ex: number, ey: number): Image;
    /**
     * resize image
     * 
     * @param width width
     * @param height height, if not set, keep aspect ratio
     * @return resized image
     */
    resize(width: number, height?: number): Image;
    /**
     * lasso tool to replace colors within the lassoed area
     * 
     * @param points points of the lasso
     * @param src source color with optional tolerance
     * @param dst destination color
     * @return modified image
     */
    lasso(points: [x: number, y: number][], src: [r: number, g: number, b: number, a: number, rt?: number, gt?: number, bt?: number, at?: number], dst: [r: number, g: number, b: number, a: number]): Image;
    /**
     * to JPG format
     * 
     * @param quality quality from 1 to 100, default is 80
     * @return JPG buffer
     */
    toJPG(quality?: number): Buffer;
    /**
     * to PNG format
     * 
     * @return PNG buffer
     */
    toPNG(): Buffer;
}
declare function $native(name: "image"): {
    /**
     * create a blank image with given width and height
     * 
     * @param width width
     * @param height height
     * @return image
     */
    create(width: number, height: number): Image;
    /**
     * parse image from input data
     * 
     * @param input input data
     * @return image
     */
    parse(input: GenericByteArray): Image;
}

declare function $native(name: "lock"): (name: string) => {
    /**
     * lock with timeout
     * 
     * @param timeout timeout in milliseconds
     * @return void
     */
    lock(timeout: number): void;
    /**
     * unlock
     * 
     * @return void
     */
    unlock(): void;
}

declare function $native(name: "pipe"): (name: string) => BlockingQueue;

type TCPSocketConnection = {
    /**
     * read data from the connection
     * 
     * @param size size of data to read
     * @return data buffer
     */
    read(size?: number): Buffer;
    /**
     * read a line from the connection
     * 
     * @return line buffer
     */
    readLine(): Buffer;
    /**
     * write data to the connection
     * 
     * @param data data to write
     * @return number of bytes written
     */
    write(data: GenericByteArray): number;
    /**
     * close the connection
     * 
     * @return void
     */
    close(): void;
}
type UDPSocketConnection = {
    /**
     * read data from the connection
     * 
     * @param size size of data to read
     * @return data buffer
     */
    read(size?: number): Buffer;
    /**
     * write data to the connection
     * 
     * @param data data to write
     * @param host host
     * @param port port
     * @return number of bytes written
     */
    write(data: GenericByteArray, host?: string, port?: number): number;
    /**
     * close the connection
     * 
     * @return void
     */
    close(): void;
}
declare function $native(name: "socket"): {
    (protocol: "tcp"): {
        /**
         * dial to a tcp server
         * 
         * @param host host
         * @param port port
         * @return tcp socket connection
         */
        dial(host: string, port: number): TCPSocketConnection;
        /**
         * listen on a tcp port
         * 
         * @param port port
         * @return listener with accept method
         */
        listen(port: number): {
            /**
             * accept a tcp connection
             * 
             * @return tcp socket connection
             */
            accept(): TCPSocketConnection;
        };
    };
    (protocol: "udp"): {
        /**
         * dial to a udp server
         * 
         * @param host host
         * @param port port
         * @return udp socket connection
         */
        dial(host: string, port: number): UDPSocketConnection;
        /**
         * listen on a udp port
         * 
         * @param port port
         * @return udp socket connection
         */
        listen(port: number): UDPSocketConnection;
        /**
         * listen on a udp multicast address
         * 
         * @param host host
         * @param port port
         * @return udp socket connection
         */
        listenMulticast(host: string, port: number): UDPSocketConnection;
    };
}

declare function $native(name: "process"): {
    /**
     * execute a command with parameters, return output buffer
     * 
     * @param command command
     * @param params parameters
     * @return output buffer
     */
    exec(command: string, ...params: string[]): Buffer;
    /**
     * execute a command with parameters asynchronously, return output buffer
     * 
     * @param command command
     * @param params parameters
     * @return promise of output buffer
     */
    pexec(command: string, ...params: string[]): Promise<Buffer>;
};

declare function $native(name: "template"): (name: string, input: { [name: string]: any; }) => string;

declare function $native(name: "ulid"): () => string;

type XmlNode = {
    /**
     * find nodes by xpath expression
     * 
     * @param expr expression
     * @return array of xml nodes
     */
    find(expr: string): XmlNode[];
    /**
     * find one node by xpath expression
     * 
     * @param expr expression
     * @return xml node
     */
    findOne(expr: string): XmlNode;
    /**
     * inner text of this node
     * 
     * @return inner text
     */
    innerText(): string;
    /**
     * to string
     * 
     * @return string
     */
    toString(): string;
}
declare function $native(name: "xml"): (content: string) => XmlNode;

type ZipEntry = {
    /**
     * name of the entry
     */
    name: string;
    /**
     * compressed size of the entry
     */
    compressedSize64: number;
    /**
     * uncompressed size of the entry
     */
    uncompressedSize64: number;
    /**
     * comment of the entry
     */
    comment: string;
    /**
     * get data of the entry
     * 
     * @return data buffer
     */
    getData(): Buffer;
}
declare function $native(name: "zip"): {
    /**
     * write data to zip format
     * 
     * @param data data object with name and content
     * @return zip buffer
     */
    write(data: { [name: string]: string | Buffer; }): Buffer;
    /**
     * read zip data
     * 
     * @param data data in zip format
     * @return zip reader object
     */
    read(data: GenericByteArray): {
        /**
         * get all entries in the zip
         * 
         * @return array of zip entries
         */
        getEntries(): ZipEntry[];
        /**
         * get entry by name
         * 
         * @param name name of the entry
         * @return zip entry
         */
        getData(name: string): Buffer;
    };
}

//#endregion


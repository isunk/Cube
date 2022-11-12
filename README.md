# Cube

A lightweight web server enabling online development with TypeScript/JavaScript.

## Getting Started

1. Clone the repository.

2. Build from source:
    ```bash
    make build
    ```
    Or build with CDN and UPX compression:
    ```bash
    make build CDN=1 UPX=1
    ```

3. Start the server:
    ```bash
    ./cube -n 256
    ```
    Alternatively, run directly from source:
    ```bash
    make run
    ```
    For additional startup parameters:
    ```bash
    ./cube --help
    ```

4. Open `http://127.0.0.1:8090/` in your browser.

### Run with SSL/TLS

1. Generate `ca.key`, `ca.crt`, `server.key`, and `server.crt`:
    ```bash
    make crt
    ```

2. Start the server:
    ```bash
    ./cube \
        -n 256 \ # allocate 256 virtual machines
        -p 8443 \ # bind to port 8443
        -s \ # enable SSL/TLS
        -v # enable client certificate verification
    ```

3. For self-signed certificates, install `ca.crt` to the system root certificate store:
    ```cmd
    rem Install ca.crt into the Trusted Root Certification Authorities store
    certutil -addstore root ca.crt
    ```

4. Access the server at `https://127.0.0.1:8443/` in your browser.

5. Test with client certificate authentication using curl:
    ```bash
    # Generate client.key and client.crt
    make ccrt

    # Authenticate with client certificate
    curl --cacert ./ca.crt --cert ./client.crt --key ./client.key https://127.0.0.1:8443/service/foo
    ```
    Or use Chrome with a client certificate:
    ```cmd
    rem Convert client.crt and client.key to PKCS#12 format
    openssl pkcs12 -export -in client.crt -inkey client.key -out client.p12 -passout pass:123456

    rem Import client.p12 into the Personal certificate store
    certutil -importPFX -f -p 123456 My client.p12

    rem Launch Chrome and select the client certificate
    chrome https://127.0.0.1:8443/
    ```

### Run with HTTP/3

1. Generate SSL certificates as described above:
    ```bash
    make crt
    ```

2. Start the server with HTTP/3 enabled:
    ```bash
    ./cube \
        -n 256 \ # allocate 256 virtual machines
        -p 8443 \ # bind to port 8443
        -s \ # enable SSL/TLS
        -3 # enable HTTP/3
    ```

3. Test with curl:
    ```bash
    curl --http3 -I https://127.0.0.1:8443/service/foo
    ```
    Or test with Chrome using QUIC:
    ```cmd
    rem Terminate all running Chrome instances
    taskkill /f /t /im chrome.exe

    rem Launch Chrome with QUIC enabled for the target origin
    chrome --enable-quic --origin-to-force-quic-on=127.0.0.1:8443 https://127.0.0.1:8443/
    ```

### Run on Termux

1. Download the latest release:
    ```bash
    wget https://xget.xi-xu.me/gh/isunk/Cube/releases/download/latest/cube-latest-linux-arm64.tar.gz && tar zxvf cube-latest-linux-arm64.tar.gz
    ```

2. Configure and start the server:
    ```bash
    # Install proot
    pkg install proot

    # Create and link the database file
    touch ~/storage/shared/cube.db && ln -s ~/storage/shared/cube.db ~/cube.db

    # Link the files directory
    ln -s ~/storage/shared/Download ~/files

    # Set execution permissions
    chmod +x ~/cube

    # Run with required system file bindings
    proot -b $PREFIX/etc/resolv.conf:/etc/resolv.conf -b $PREFIX/etc/tls/cert.pem:/etc/ssl/certs/ca-certificates.crt ./cube
    ```

## Examples

### Controller

Controllers handle HTTP/HTTPS requests and return responses.

- Basic controller:
    ```typescript
    export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
        return "hello, world"
    }
    ```

- Access request parameters:
    1. Create a controller named `greeting` with type `controller` and URL `/service/{name}/greeting/{words}`:
        ```typescript
        export default function (ctx: ServiceContext) {
            // Retrieve request body as string
            String.fromCharCode(...ctx.getBody())

            // Extract path variables
            ctx.getPathVariables() // {"name":"zhangsan","words":"hello"}

            // Parse form data
            ctx.getForm() // {"a":["1","3"],"b":["2"],"c":[""],"d":["4","6"],"e":["5"],"f":[""]}

            // Get URL path and query parameters
            ctx.getURL() // {"params":{"a":["1","3"],"b":["2"],"c":[""]},"path":"/service/foo"}
        }
        ```
    2. Test with curl:
        ```bash
        curl -XPOST -H "Content-Type: application/x-www-form-urlencoded" "http://127.0.0.1:8090/service/zhangsan/greeting/hello?a=1&b=2&c&a=3" -d "d=4&e=5&f&d=6"
        ```

- Return custom responses:
    ```typescript
    export default function (ctx: ServiceContext): ServiceResponse {
        // return new Uint8Array([104, 101, 108, 108, 111]) // response with body "hello"
        return new ServiceResponse(500, {
            "Content-Type": "text/plain",
        }, new Uint8Array([104, 101, 108, 108, 111]))
    }
    ```

- WebSocket server:
    ```typescript
    export default function (ctx: ServiceContext) {
        const ws = ctx.upgradeToWebSocket() // upgrade HTTP connection to WebSocket
        console.info(ws.read()) // read incoming message
        ws.send("hello, world") // send message to client
        ws.close() // terminate connection
    }
    ```

- HTTP chunked transfer:
    1. Create a controller named `foo` with type `controller` and URL `/service/foo`:
        ```typescript
        export default function (ctx: ServiceContext) {
            ctx.write("hello, chunk 0")
            ctx.flush()
            ctx.write("hello, chunk 1")
            ctx.flush()
            ctx.write("hello, chunk 2")
            ctx.flush()
        }
        ```
    2. Test with telnet:
        ```bash
        { echo "GET /service/foo HTTP/1.1"; echo "Host: 127.0.0.1"; echo ""; sleep 1; echo exit; } | telnet 127.0.0.1 8090
        ```

- Read request body incrementally:
    ```typescript
    export default function (ctx: ServiceContext) {
        const reader = ctx.getReader()

        // String.fromCharCode(...reader.read(10)) // Read 10 bytes from request body. Returns null on EOF.

        const arr = []

        let byte = reader.readByte()
        while (byte != -1) { // Returns -1 on EOF
            arr.push(byte)
            byte = reader.readByte()
        }

        console.debug(String.fromCharCode(...arr))
    }
    ```

### Module

Modules provide reusable code that can be imported by controllers.

- Custom module:
    ```typescript
    export const user = {
        name: "zhangsan"
    }
    ```
    ```typescript
    import { user } from "./user"

    export default function (ctx: ServiceContext) {
        return `hello, ${user?.name ?? "world"}`
    }
    ```

- [Extend the Number prototype](docs/modules/number.md)

- Import modules from remote sources via HTTP/HTTPS:
    ```typescript
    import * as JSON5 from "https://unpkg.com/json5@2/dist/index.min.js"

    export default function (ctx: ServiceContext) {
        return JSON5.parse("{ greeting: 'hello, world' }")
    }
    ```

### Daemon

Daemons are long-running background services with no execution timeout.

- Create a daemon:
    ```typescript
    export default function () {
        const b = $native("pipe")("default")
        while (true) {
            console.info(b.drain(100, 5000))
        }
    }
    ```

### Built-in Methods and Modules

The following built-in utilities are available:

- Buffer:
    ```typescript
    const buf = Buffer.from("hello", "utf8")
    buf // [104, 101, 108, 108, 111]
    buf.toString("base64") // aGVsbG8=
    String.fromCharCode(...buf) // hello
    ```

- Console:
    ```typescript
    // ...
    console.error("this is an error message")
    ```

- Date:
    ```typescript
    Date.toDate("2006-01-02 15:04:05.012", "yyyy-MM-dd HH:mm:ss.SSS")
        .toString("yyyyMMddHHmmssSSS") // "20060102150405012"
    ```

- Decimal:
    ```typescript
    const d1 = new Decimal("0.1"),
        d2 = new Decimal("0.2")
    d2.add(d1) // 0.3
    d2.sub(d1) // 0.1
    d2.mul(d1) // 0.02
    d2.div(d1) // 2
    ```

- Error Handling:
    ```typescript
    // ...
    throw new Error("error message")

    // ...
    throw {
        code: "error code",
        message: "error message"
    }
    ```

### Native Modules

Native modules provide access to system-level functionality:

- Bqueue & Pipe:
    ```typescript
    const b = $native("pipe")("mypipe")
    // const b = $native("bqueue")(99)
    b.put(1)
    b.put(2)
    b.drain(4, 2000) // [1, 2]
    ```

- Database:
    ```typescript
    $native("db").query("select name from script") // [{"name":"foo"}, {"name":"user"}]
    ```

- Email:
    ```typescript
    const emailc = $native("email")("smtp.163.com", 465, username, password)
    emailc.send(["zhangsan@abc.com"], "greeting", "hello, world")
    emailc.send(["zhangsan@abc.com"], "greeting", "hello, world", [{
        name: "hello.txt",
        contentType: "text/plain",
        base64: "aGVsbG8=",
    }])
    ```

- Crypto:
    ```typescript
    const cryptoc = $native("crypto")
    // hash
    cryptoc.createHash("md5").sum("hello, world").map(c => c.toString(16).padStart(2, "0")).join("") // "e4d7f1b4ed2e42d15898f4b27b019da4"
    // hmac
    cryptoc.createHmac("sha1").sum("hello, world", "123456").toString("hex") // "9a231f1dd39a4ff6ea778a5640d1498794f8a9f8"
    // rsa
    // privateKey and publicKey are in PKCS#1 format
    const rsa = cryptoc.createRsa(),
        { privateKey, publicKey } = rsa.generateKey();
    rsa.decrypt(
        rsa.encrypt("hello, world", publicKey),
        privateKey,
    ).toString() // "hello, world"
    rsa.verify(
        "hello, world",
        rsa.sign("hello, world", privateKey, "sha256", "pss"),
        publicKey,
        "sha256",
        "pss",
    ) // true
    ```

- File System:
    ```typescript
    const filec = $native("file")
    filec.write("greeting.txt", "hello, world")
    String.fromCharCode(...filec.read("greeting.txt")) // "hello, world"
    ```

- HTTP Client:
    ```typescript
    const httpc = $native("http")({
        // caCert: "",                     // CA certificates for TLS verification
        // cert: "", key: "",              // Client certificate and private key for mutual TLS
        // proxy: "http://127.0.0.1:5566", // HTTP proxy server
        // isSkipInsecureVerify: true,     // Disable server certificate validation
        // isHttp3: true,                  // Enable HTTP/3 support
        // isNotFollowRedirect: true,      // Disable automatic redirect following
    })
    const { status, header, data } = httpc.request("GET", "https://www.baidu.com")
    status // 200
    header // { "Content-Length": "227", "Content-Type": "text/html", ... }
    data.toString() // "<html>..."
    ```

- Image Processing:
    ```typescript
    const imagec = $native("image"),
        filec = $native("file")

    const img = imagec.parse(filec.read("input.jpg")),
        text = "hello, world",
        textHeight = 28,
        textWidth = text.length * textHeight * 0.46,
        rotation = -30

    img.setDrawFontFace(textHeight)
    img.setDrawColor([255, 255, 255, 80])
    img.setDrawRotate(rotation)

    for (let i = 0, di = textWidth / Math.tan(Math.PI / 180 * rotation * -1), ic = img.width() / di; i < ic; i++) {
        for (let j = 0, dj = textWidth, jc = img.height() / dj; j < jc; j++) {
            img.drawString(text, i * di + 20, j * dj)
        }
    }

    img.drawString(text, img.width(), img.height() - textHeight, 1, 1) // write text in the bottom right corner

    // Use the lasso tool to replace pixels in the specified range with transparency
    // Pixels ranging from [0, 0, 0, 255] to [200, 200, 200, 255] are replaced with [0, 0, 0, 0] within the polygonal area defined by [(0, 0), (1200, 0), (1200, 1200)]
    const img2 = img.lasso([
        [0, 0], [1200, 0], [1200, 1200],
    ], [0, 0, 0, 255, 200, 200, 200], [0, 0, 0, 0])

    filec.write("output.jpg", img2.resize(1280).toJPG())
    ```

- Template Engine:
    ```typescript
    const content = $native("template")("greeting", { // read template greeting.tpl and render with data
        name: "this is name",
    })
    ```

- XML Processing:
    ```typescript
    // XPath syntax follows https://github.com/antchfx/xpath
    const doc = $native("xml")(`
        <Users>
            <User>
                <ID>1</ID>
                <Name>zhangsan</Name>
            </User>
            <User>
                <ID>2</ID>
                <Name>lisi</Name>
            </User>
        </Users>
    `)
    doc.find("//user[id=2]/name").pop().innerText() // lisi
    doc.findOne("//user[1]/name").innerText() // zhangsan
    doc.findOne("//user[1]").findOne("name").innerText() // zhangsan
    ```

### Advanced Topics

- File Upload:
    1. Create a resource with language `html` and URL `/resource/foo.html`:
        ```html
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="UTF-8">
            <link rel="stylesheet" href="//unpkg.com/element-ui/lib/theme-chalk/index.css">
        </head>
        <body>
            <div id="app" v-cloak>
                <el-upload
                    action="/service/foo"
                    accept="image/jpeg"
                    :auto-upload="true">
                    <el-button icon="el-icon-upload2">Upload</el-button>
                </el-upload>
            </div>
        </body>
        <script src="//cdnjs.cloudflare.com/ajax/libs/vue/2.7.14/vue.js"></script>
        <script src="//unpkg.com/element-ui"></script>
        <script>
            new Vue({ el: "#app" })
        </script>
        </html>
        ```
    2. Create a controller at `/service/foo`:
        ```typescript
        export default function (ctx: ServiceContext) {
            const file = ctx.getFile("file"),
                hash = $native("crypto").createHash("md5").sum(file.data).toString("hex")
            console.info(hash)
        }
        ```
    3. Preview at `http://127.0.0.1:8090/resource/foo.html`, or test with curl:
        ```bash
        # Upload a file
        curl -F "file=@./abc.txt; filename=abc.txt;" http://127.0.0.1:8090/service/foo
        ```

### Additional Resources

For more examples and detailed documentation, refer to the [documentation summary](docs/summary.md).
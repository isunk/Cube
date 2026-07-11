# Documentation

Welcome to the Cube documentation. Below you'll find comprehensive guides and examples for leveraging Cube's capabilities.

## Modules

### Data & Database

- **[SQL](modules/sql.md)** - SQL query builder with template syntax supporting conditional clauses and parameter binding
- **[Linq2SQL](modules/linq2sql.md)** - LINQ-style query API for type-safe database operations
- **[GraphQL](modules/graphql.md)** - GraphQL-like query and mutation engine with schema-based data access
- **[DbHelper](modules/dbhelper.md)** - Database abstraction layer supporting MySQL and SQLite with unified CRUD operations

### Workflow & Process

- **[BPM](modules/bpm.md)** - Business Process Management engine with support for user tasks, gateways, and process orchestration
- **[WebFlow](modules/webflow.md)** - Web application flow controller managing state transitions between views and actions

### Validation & Security

- **[Validator](modules/validator.md)** - Schema-based parameter validation supporting strings, numbers, dates, objects, and collections
- **[XSS](modules/xss.md)** - Cross-site scripting (XSS) filter with configurable tag and attribute whitelist
- **[Permission](modules/permission.md)** - Role-based access control with wildcard pattern matching for permission checks

### Utilities

- **[Number](modules/number.md)** - Extended Number prototype with IPv4 address conversion utilities
- **[CSV](modules/csv.md)** - CSV parser and serializer supporting complex data formats and custom delimiters
- **[SSR](modules/ssr.md)** - Server-side rendering engine with template syntax and client-server method binding

## Examples

### Web Development

- **[Vue Integration](examples/vue.md)** - Asynchronous Vue component loading with Vue 2 and Vue 3 support
- **[Dynamic Views](examples/dynamic-views.md)** - Dynamic view rendering with runtime component loading

### Media & Streaming

- **[HTTP-FLV Streaming](examples/httpflv.md)** - Live video streaming using HTTP-FLV protocol with flv.js
- **[DLNA Casting](examples/dlna.md)** - Media casting to DLNA-enabled devices on local network
- **[Video-on-Demand](examples/vodd.md)** - VOD server with MAC CMS integration and content crawling
- **[RTMP Server](examples/rtmpd.md)** - RTMP streaming server with HTTP-FLV conversion

### Servers & Services

- **[HTTP Server](examples/httpd.md)** - Multi-purpose HTTP server with image resizing, MP4 range requests, and ZIP preview
- **[WebDAV Server](examples/webdavd.md)** - WebDAV protocol implementation for file management
- **[SMTP Server](examples/smtpd.md)** - SMTP server implementation using socket module
- **[Mock Server](examples/mockd.md)** - Mock API server for development and testing

### Authentication & Payments

- **[CAS SSO](examples/casd.md)** - Single Sign-On server based on CAS protocol
- **[Alipay Payment](examples/alipay.md)** - Online payment integration with Alipay sandbox

### Real-time Communication

- **[WebRTC Chat](examples/webrtc.md)** - Peer-to-peer web chat application using PeerJS
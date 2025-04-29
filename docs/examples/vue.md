# Return a view with asynchronous vues

## Vue 2

1. Create a template with lang `html` and name `Vue2App`.
    ```html
    <!DOCTYPE html>
    <html>
    <head>
        <meta charset="utf-8" />
        <title>{{ .title }}</title>
        <style>
            * {
                margin: 0;
                padding: 0;
            }
            html, body {
                width: 100%;
                height: 100%;
            }
            html {
                overflow: hidden;
            }
        </style>
    </head>
    <body>
        <script src="https://cdn.bootcdn.net/ajax/libs/vue/2.7.14/vue.min.js"></script>
        <script src="https://cdn.bootcdn.net/ajax/libs/vue-router/3.6.5/vue-router.min.js"></script>
        <script src="https://unpkg.com/http-vue-loader"></script>
        <router-view id="container"></router-view>
        <script>
            const router = new VueRouter({
                mode: "hash"
            })
            router.beforeEach((to, from, next) => {
                if (to.matched.length) { // 当前路由已匹配上
                    next() // 直接渲染当前路由
                    return
                }
                router.addRoute({ // 动态添加路由
                    path: to.path,
                    component: httpVueLoader(`../resource${to.path === "/" ? "/index" : to.path}.vue`), // 远程加载组件
                })
                next(to.path) // 重新进入 beforeEach 方法
            })
            new Vue({ router }).$mount("#container")
        </script>
    </body>
    </html>
    ```

2. Create a resource with lang `vue` and url `/resource/greeting.vue`.
    ```html
    <template>
        <p>hello, {{ name }}</p>
    </template>

    <script>
        module.exports = {
            data: function() {
                return {
                    name: "world",
                }
            }
        }
    </script>

    <style scoped>
        p {
            color: #000;
        }
    </style>
    ```

3. Create a controller with url `/service/app/vue2`.
    ```typescript
    export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
        return $native("template")("Vue2App", {
            title: "this is title",
        })
    }
    ```

4. You can preview at `http://127.0.0.1:8090/service/app/vue2#/greeting`

## Vue 3

1. Create a resource with lang `html` and url `/resource/vue3/app`.
    ```html
    <!DOCTYPE html>
    <html>
    <body>
        <script src="https://unpkg.com/vue@3.4.6/dist/vue.global.prod.js"></script>
        <script src="https://unpkg.com/vue-router@4.5.1/dist/vue-router.global.prod.js"></script>
        <script src="https://cdn.jsdelivr.net/npm/vue3-sfc-loader@0.9.5/dist/vue3-sfc-loader.js"></script>
        <script>
            const load = (() => {
                const options = {
                    moduleCache: {
                        vue: Vue,
                        router: VueRouter,
                    },
                    async getFile(url) {
                        const res = await fetch(url);
                        if (!res.ok) {
                            throw Object.assign(new Error(res.statusText + " " + url), { res });
                        }
                        return {
                            getContentData: (asBinary) => asBinary ? res.arrayBuffer() : res.text(),
                        };
                    },
                    addStyle(textContent) {
                        const style = Object.assign(document.createElement("style"), { textContent }),
                            ref = document.head.getElementsByTagName("style")[0] || null;
                        document.head.insertBefore(style, ref);
                    },
                    log: (...args) => console.log(...args),
                };
                return (name) => Vue.defineAsyncComponent(() => window["vue3-sfc-loader"].loadModule(name, options));
            })();

            const router = VueRouter.createRouter({
                history: VueRouter.createWebHashHistory(),
                routes: [
                    { path: "/", component: load("/resource/greeting.vue"), },
                ],
            });

            Vue.createApp({
                template: "<router-view />",
            }).use(router).mount(document.body);
        </script>
    </body>
    </html>
    ```

2. You can preview at `http://127.0.0.1:8090/resource/app/vue3`

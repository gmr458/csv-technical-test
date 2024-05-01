import {
    Application,
    Router,
    Status,
} from "https://deno.land/x/oak@v16.0.0/mod.ts";

let globalData: Record<string, string>[];

const router = new Router();

router
    .post("/api/files", async (context) => {
        let formData: FormData;
        try {
            formData = await context.request.body.formData();
        } catch (err: unknown) {
            if (err instanceof TypeError) {
                context.response.status = Status.BadRequest;
                context.response.body = { message: "A file must be provided" };
                return;
            }

            context.response.status = Status.InternalServerError;
            context.response.body = { message: "Internal server error" };
            return;
        }

        const file = formData.get("file");
        if (!file) {
            context.response.status = Status.BadRequest;
            context.response.body = { message: "A file must be provided" };
        }

        if (file instanceof File === false) {
            context.response.status = Status.BadRequest;
            context.response.body = { message: "A file must be provided" };
            return;
        }

        if (file.type !== "text/csv") {
            context.response.status = Status.BadRequest;
            context.response.body = { message: "The file type must be CSV" };
            return;
        }

        const text = await file.text();
        if (text.length === 0) {
            context.response.status = Status.BadRequest;
            context.response.body = { message: "The file must not be empty" };
            return;
        }

        const data = text.trim().split("\n");
        if (data.length < 2) {
            context.response.status = Status.BadRequest;
            context.response.body = { message: "Send a file with records" };
            return;
        }

        const [keys, ...rows] = data.map((line) => line.split(","));

        if (globalData && globalData.length > 0) {
            globalData = [];
        }

        globalData = rows.map((cell) => {
            const item: Record<string, string> = {};
            keys.forEach((key, index) => {
                item[key] = cell[index];
            });
            return item;
        });

        context.response.status = Status.OK;
        context.response.body = { message: "File uploaded successfully" };
    })
    .get("/api/users", (context) => {
        if (!globalData) {
            context.response.body = {
                message: "There is not data, upload a CSV file first",
            };
            return;
        }

        let q = context.request.url.searchParams.get("q");
        if (!q) {
            context.response.body = {
                data: globalData,
            };
            return;
        }

        q = q.toLowerCase();
        const coincidences: Record<string, string>[] = [];

        for (const record of globalData) {
            for (const key in record) {
                if (record[key].toLowerCase().indexOf(q) !== -1) {
                    coincidences.push(record);
                }
            }
        }

        context.response.body = {
            data: coincidences,
        };
    });

const app = new Application();

app.use(async (ctx, next) => {
    await next();
    const rt = ctx.response.headers.get("X-Response-Time");
    console.log(`${ctx.request.method} ${ctx.request.url} - ${rt}`);
});

app.use(async (ctx, next) => {
    const start = Date.now();
    await next();
    const ms = Date.now() - start;
    ctx.response.headers.set("X-Response-Time", `${ms}ms`);
});

app.use(router.routes());
app.use(router.allowedMethods());

const port = 3000;

console.log(`server is running at http://localhost:${port}`);
await app.listen({ port });

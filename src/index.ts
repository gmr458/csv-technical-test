import { Elysia, t } from "elysia";
import { swagger } from "@elysiajs/swagger";

let globalData: Record<string, string>[];

const app = new Elysia()
	.use(swagger())
	.post(
		"/api/files",
		async (c) => {
			if (!c.body.file) {
				return c.error(400, { message: "A file should be provided" });
			}

			if (c.body.file.type !== "text/csv") {
				return c.error(400, { message: "File type should be CSV" });
			}

			const text = await c.body.file.text();
			if (text.length === 0) {
				return c.error(400, { message: "File must not be empty" });
			}

			const data = text.trim().split("\n");
			if (data.length < 2) {
				return c.error(400, { message: "File must not be empty" });
			}

			const [keys, ...rows] = data.map((line) => line.split(","));

			globalData = rows.map((cell) => {
				const item: Record<string, string> = {};
				keys.forEach((key, index) => {
					item[key] = cell[index];
				});
				return item;
			});

			return { message: "File uploaded successfully" };
		},
		{
			body: t.Object({
				file: t.Optional(t.File()),
			}),
			type: "multipart/form-data",
		},
	)
	.get(
		"/api/users",
		async (c) => {
			if (!globalData) {
				return {
					message: "There is not data, send a file to /api/files",
				};
			}

			if (!c.query.q) {
				return {
					data: globalData,
				};
			}

			const q = c.query.q.toLowerCase();
			const coincidences: Record<string, string>[] = [];

			for (const record of globalData) {
				for (const key in record) {
					if (record[key].toLowerCase().indexOf(q) !== -1) {
						coincidences.push(record);
					}
				}
			}

			return { data: coincidences };
		},
		{
			query: t.Object({
				q: t.Optional(t.String()),
			}),
		},
	);

const port = 3000;

app.listen(port);

console.log(`server is running at http://localhost:${port}`);

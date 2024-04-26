import { Elysia, t } from "elysia";
import { swagger } from "@elysiajs/swagger";

let globalData: string[][] = [[]];

const app = new Elysia()
	.use(swagger())
	.post(
		"/api/files",
		async (c) => {
			console.log(c.body);

			if (!c.body.file) {
				return c.error(400, { message: "A file should be provided" });
			}

			if (c.body.file.type !== "text/csv") {
				return c.error(400, { message: "File type should be CSV" });
			}

			const text = await c.body.file.text();
			const data = text
				.split("\n")
				.filter((line) => line.length !== 0)
				.map((line) => line.split(",").map((word) => word.trim()));
			globalData = data;

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
			if (!c.query.q) {
				return {
					data: globalData,
				};
			}

			const q = c.query.q.toLowerCase();
			const keys = globalData[0];
			const rows = globalData.slice(1);

			const coincidences: Record<string, string>[] = [];

			for (const row of rows) {
				for (const cell of row) {
					if (cell.toLowerCase().indexOf(q) !== -1) {
						const item: Record<string, string> = {};

						for (let index = 0; index < keys.length; index++) {
							const value = row[index];
							const key = keys[index];
							item[key] = value;
						}

						coincidences.push(item);
					}
				}
			}

			if (coincidences.length === 0) {
				return c.error(404, { message: "Not found" });
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
